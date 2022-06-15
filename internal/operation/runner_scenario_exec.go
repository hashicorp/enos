package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// func ExecScenario takes an operation request for generate and returns a worker
// function to generate a terraform module for a scenario.
func ExecScenario(req *pb.Operation_Request) WorkFunc {
	return func(
		ctx context.Context,
		eventC chan *pb.Operation_Event,
		log hclog.Logger,
	) *pb.Operation_Response {
		events := NewEventSender(eventC)

		resVal := &pb.Operation_Response_Exec_{
			Exec: &pb.Operation_Response_Exec{},
		}

		// Create our new response from our request
		res, err := NewResponseFromRequest(req)
		if err != nil {
			log.Debug("failed to create response", RequestDebugArgs(req)...)
			if err = events.PublishResponse(res); err != nil {
				log.Error("failed to send event", ResponseDebugArgs(res)...)
			}
			return res
		}

		res.Value = resVal

		runner := NewRunner(
			WithRunnerTerraformConfig(req.GetWorkspace().GetTfExecCfg()),
			WithLogger(log),
		)

		resVal.Exec.Exec = runner.terraformExec(ctx, req, events)
		res.Status = diagnostics.Status(runner.TFConfig.FailOnWarnings, resVal.Exec.Exec.GetDiagnostics()...)

		return res
	}
}
