// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	"github.com/hashicorp/go-hclog"
	tfjson "github.com/hashicorp/terraform-json"
)

// RunScenario takes an operation request for generate and returns a worker
// function to generate a terraform module for a scenario.
func RunScenario(req *pb.Operation_Request) WorkFunc {
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
			if err = events.PublishResponse(res); err != nil {
				log.Error("failed to send event", "error", err)
			}

			return res
		}

		runner := NewRunner(
			WithRunnerTerraformConfig(req.GetWorkspace().GetTfExecCfg()),
			WithLogger(log),
		)

		// Create our response value
		resVal := &pb.Operation_Response_Run_{
			Run: &pb.Operation_Response_Run{},
		}
		res.Value = resVal

		// Generate our module
		genVal := runner.moduleGenerate(ctx, req, events).Generate
		resVal.Run.Generate = genVal

		// Determine our status
		res.Status = diagnostics.Status(runner.TFConfig.FailOnWarnings, genVal.GetDiagnostics()...)

		// Return early if we failed to generate our module
		if hasFailedStatus(res.GetStatus()) {
			if err := events.PublishResponse(res); err != nil {
				res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
				log.Error("failed to send event", "error", err)
			}

			return res
		}

		// Run the scenario
		run := runner.scenarioRun(ctx, req, events).Run
		resVal.Run.Diagnostics = run.GetDiagnostics()
		resVal.Run.Init = run.GetInit()
		resVal.Run.Validate = run.GetValidate()
		resVal.Run.Plan = run.GetPlan()
		resVal.Run.Apply = run.GetApply()
		resVal.Run.Destroy = run.GetDestroy()

		// Determine our final status from all operations
		res.Status = diagnostics.OperationStatus(runner.TFConfig.FailOnWarnings, res)

		return res
	}
}

// scenarioRun initializes, validates, plans, applies and destroys the generatedTerraform
// Terraform module.
func (r *Runner) scenarioRun(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
) *pb.Operation_Response_Run_ {
	// Create the run response value
	res := &pb.Operation_Response_Run_{
		Run: &pb.Operation_Response_Run{},
	}

	// launch the Terraform module
	launchRes := r.scenarioLaunch(ctx, req, events)

	res.Run.Diagnostics = launchRes.Launch.GetDiagnostics()
	res.Run.Init = launchRes.Launch.GetInit()
	res.Run.Validate = launchRes.Launch.GetValidate()
	res.Run.Plan = launchRes.Launch.GetPlan()
	res.Run.Apply = launchRes.Launch.GetApply()

	// Return early if we failed to apply our module
	if diagnostics.HasFailed(
		r.TFConfig.FailOnWarnings,
		res.Run.GetDiagnostics(),
		res.Run.GetInit().GetDiagnostics(),
		res.Run.GetValidate().GetDiagnostics(),
		res.Run.GetPlan().GetDiagnostics(),
		res.Run.GetApply().GetDiagnostics(),
	) {
		return res
	}

	// Get the current state of the scenario because destroying requires it
	stateVal := r.terraformShow(ctx, req, events)
	res.Run.PriorStateShow = stateVal

	state := &tfjson.State{}
	err := state.UnmarshalJSON(stateVal.GetState())
	if err != nil {
		stateVal.Diagnostics = append(stateVal.GetDiagnostics(), diagnostics.FromErr(err)...)
	}

	if diagnostics.HasFailed(r.TFConfig.FailOnWarnings, stateVal.GetDiagnostics()) {
		return res
	}

	res.Run.Destroy = r.terraformDestroy(ctx, req, events, state)

	return res
}
