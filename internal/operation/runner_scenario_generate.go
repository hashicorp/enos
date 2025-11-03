// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	"github.com/hashicorp/go-hclog"
)

// GenerateScenario takes an operation request for generate and returns a worker
// function to generate a terraform module for a scenario.
func GenerateScenario(req *pb.Operation_Request) WorkFunc {
	return func(
		ctx context.Context,
		eventC chan *pb.Operation_Event,
		log hclog.Logger,
	) *pb.Operation_Response {
		log = log.With(RequestDebugArgs(req)...)
		events := NewEventSender(eventC)

		// Create our new response from our request.
		res, err := NewResponseFromRequest(req)
		if err != nil {
			log.Debug("failed to create response")
			if err := events.PublishResponse(res); err != nil {
				res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
				log.Error("failed to send event", "error", err)
			}

			return res
		}

		runner := NewRunner(
			WithRunnerTerraformConfig(req.GetWorkspace().GetTfExecCfg()),
			WithLogger(log),
		)

		// Run generate and update the generic status
		resVal := runner.moduleGenerate(ctx, req, events)
		res.Value = resVal

		// Determine our final status from all operations
		res.Status = diagnostics.OperationStatus(runner.TFConfig.FailOnWarnings, res)

		return res
	}
}
