// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	"github.com/hashicorp/go-hclog"
)

// CheckScenario takes an operation request for check and returns a worker
// function to checks a scenario terraform module.
func CheckScenario(req *pb.Operation_Request) WorkFunc {
	return func(
		ctx context.Context,
		eventC chan *pb.Operation_Event,
		log hclog.Logger,
	) *pb.Operation_Response {
		log = log.With(RequestDebugArgs(req)...)
		events := NewEventSender(eventC)

		runner := NewRunner(
			WithRunnerTerraformConfig(req.GetWorkspace().GetTfExecCfg()),
			WithLogger(log),
		)

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

		// Create our response value
		resVal := &pb.Operation_Response_Check_{
			Check: &pb.Operation_Response_Check{},
		}
		res.Value = resVal

		// Generate our module
		genVal := runner.moduleGenerate(ctx, req, events).Generate
		resVal.Check.Generate = genVal

		// Determine our status
		res.Status = diagnostics.Status(runner.TFConfig.FailOnWarnings, genVal.GetDiagnostics()...)

		// Return early if we failed to generate our module
		if hasFailedStatus(res.GetStatus()) {
			return res
		}

		// Check the module
		check := runner.scenarioCheck(ctx, req, events).Check
		resVal.Check.Diagnostics = check.GetDiagnostics()
		resVal.Check.Init = check.GetInit()
		resVal.Check.Validate = check.GetValidate()
		resVal.Check.Plan = check.GetPlan()

		// Determine our final status from all operations
		res.Status = diagnostics.OperationStatus(runner.TFConfig.FailOnWarnings, res)

		return res
	}
}

// scenarioCheck initializes, validates and plans the generated Terraform module.
func (r *Runner) scenarioCheck(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
) *pb.Operation_Response_Check_ {
	// Create the check response value
	res := &pb.Operation_Response_Check_{
		Check: &pb.Operation_Response_Check{},
	}

	// initialize our Terraform module
	res.Check.Init = r.terraformInit(ctx, req, events)

	// Return early if we failed to initialize our module
	if diagnostics.HasFailed(r.TFConfig.FailOnWarnings, res.Check.GetInit().GetDiagnostics()) {
		return res
	}

	// validate our Terraform module
	res.Check.Validate = r.terraformValidate(ctx, req, events)
	// Return early if we failed to plan our module
	if diagnostics.HasFailed(r.TFConfig.FailOnWarnings, res.Check.GetValidate().GetDiagnostics()) {
		return res
	}

	// plan our Terraform module
	res.Check.Plan = r.terraformPlan(ctx, req, events)

	return res
}
