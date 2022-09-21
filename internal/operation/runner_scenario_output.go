package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// func OutputScenario takes an operation request for generate and returns a worker
// function to generate a terraform module for a scenario.
func OutputScenario(req *pb.Operation_Request) WorkFunc {
	return func(
		ctx context.Context,
		eventC chan *pb.Operation_Event,
		log hclog.Logger,
	) *pb.Operation_Response {
		log = log.With(RequestDebugArgs(req)...)
		events := NewEventSender(eventC)

		resVal := &pb.Operation_Response_Output_{
			Output: &pb.Operation_Response_Output{},
		}

		// Create our new response from our request
		res, err := NewResponseFromRequest(req)
		if err != nil {
			log.Debug("failed to create response")
			if err = events.PublishResponse(res); err != nil {
				log.Error("failed to send event", "error", err)
			}
			return res
		}
		res.Value = resVal

		runner := NewRunner(
			WithRunnerTerraformConfig(req.GetWorkspace().GetTfExecCfg()),
			WithLogger(log),
		)

		mod, diags := moduleForReq(req)
		resVal.Output.Diagnostics = append(resVal.Output.GetDiagnostics(), diags...)

		res.Status = diagnostics.Status(runner.TFConfig.FailOnWarnings, resVal.Output.Output.GetDiagnostics()...)
		if diagnostics.HasFailed(runner.TFConfig.FailOnWarnings, resVal.Output.GetDiagnostics()) {
			log.Debug("failed to load Terraform module")
			if err = events.PublishResponse(res); err != nil {
				log.Error("failed to send event", "error", err)
			}
			return res
		}

		// Configure our Terraform executor to use the module we generated
		runner.TFConfig.WithModule(mod)

		// Run the output command in the context of the module that should already
		// exist
		resVal.Output.Output = runner.terraformOutput(ctx, req, events)
		res.Status = diagnostics.Status(runner.TFConfig.FailOnWarnings, resVal.Output.Output.GetDiagnostics()...)

		return res
	}
}
