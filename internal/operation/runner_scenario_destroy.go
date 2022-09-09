package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// func DestroyScenario takes an operation request for generate and returns a worker
// function to generate a terraform module for a scenario.
func DestroyScenario(req *pb.Operation_Request) WorkFunc {
	return func(
		ctx context.Context,
		eventC chan *pb.Operation_Event,
		log hclog.Logger,
	) *pb.Operation_Response {
		log = log.With(RequestDebugArgs(req)...)
		events := NewEventSender(eventC)

		resVal := &pb.Operation_Response_Destroy_{
			Destroy: &pb.Operation_Response_Destroy{},
		}

		// Create our new response from our request
		res, err := NewResponseFromRequest(req)
		if err != nil {
			log.Debug("failed to create response", "error", err)
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

		// Generate our module
		genVal := runner.moduleGenerate(ctx, req, events).Generate
		resVal.Destroy.Generate = genVal

		// Determine our status
		res.Status = diagnostics.Status(runner.TFConfig.FailOnWarnings, genVal.GetDiagnostics()...)

		// Return early if we failed to generate our module
		if hasFailedStatus(res.Status) {
			if err := events.PublishResponse(res); err != nil {
				res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
				log.Error("failed to send event", ResponseDebugArgs(res)...)
			}

			return res
		}

		// Destroy the scenario
		resVal.Destroy.Destroy = runner.terraformDestroy(ctx, req, events)
		res.Status = diagnostics.Status(runner.TFConfig.FailOnWarnings, resVal.Destroy.Destroy.GetDiagnostics()...)

		return res
	}
}
