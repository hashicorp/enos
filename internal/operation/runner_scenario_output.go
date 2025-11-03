// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
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

		// Configure the runner with the existing Terraform module. If it doesn't
		// exit there's nothing to output.
		mod, diags := moduleForReq(ctx, req)
		resVal.Output.Diagnostics = append(resVal.Output.GetDiagnostics(), diags...)

		res.Status = diagnostics.Status(runner.TFConfig.FailOnWarnings, resVal.Output.GetOutput().GetDiagnostics()...)
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

		// Determine our final status from all operations
		res.Status = diagnostics.OperationStatus(runner.TFConfig.FailOnWarnings, res)

		return res
	}
}
