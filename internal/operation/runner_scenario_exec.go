// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
		log = log.With(RequestDebugArgs(req)...)
		events := NewEventSender(eventC)

		resVal := &pb.Operation_Response_Exec_{
			Exec: &pb.Operation_Response_Exec{},
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

		// Exec is tricky in that the sub-command may or may not actually need
		// to execute in the context of a generated scenario Terraform module.
		// As such, we'll try and configure it with the module but rewrite any
		// failure diags as warnings. If the sub-command needs to the module and
		// it doesn't exist the sub-command failure will be reported.

		// Try and configure the runner with the module
		mod, diags := moduleForReq(ctx, req)

		if len(diags) > 0 {
			// Rewrite failure diags to warnings since we might not need the module
			for i := range diags {
				if diags[i].GetSeverity() == pb.Diagnostic_SEVERITY_ERROR {
					diags[i].Severity = pb.Diagnostic_SEVERITY_WARNING
				}
			}
		}

		resVal.Exec.Diagnostics = append(resVal.Exec.GetDiagnostics(), diags...)

		// Configure our Terraform executor to use module that may or may not exist
		runner.TFConfig.WithModule(mod)

		// Execute the exec command
		resVal.Exec.Exec = runner.terraformExec(ctx, req, events)

		// Determine our final status from all operations
		res.Status = diagnostics.OperationStatus(runner.TFConfig.FailOnWarnings, res)

		return res
	}
}
