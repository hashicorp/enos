package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// LaunchScenario takes an operation request for check and returns a worker
// function to checks a scenario terraform module.
func LaunchScenario(req *pb.Operation_Request) WorkFunc {
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
				res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
				log.Error("failed to send event", "error", err)
			}
			return res
		}

		runner := NewRunner(
			WithRunnerTerraformConfig(req.GetWorkspace().GetTfExecCfg()),
			WithLogger(log),
		)

		// Create our response value
		resVal := &pb.Operation_Response_Launch_{
			Launch: &pb.Operation_Response_Launch{},
		}
		res.Value = resVal

		// Generate our module
		genVal := runner.moduleGenerate(ctx, req, events).Generate
		resVal.Launch.Generate = genVal

		// Determine our status
		res.Status = diagnostics.Status(runner.TFConfig.FailOnWarnings, genVal.GetDiagnostics()...)

		// Return early if we failed to generate our module
		if hasFailedStatus(res.Status) {
			if err := events.PublishResponse(res); err != nil {
				res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
				log.Error("failed to send event", "error", err)
			}

			return res
		}

		// Launch the module
		launch := runner.scenarioLaunch(ctx, req, events).Launch
		resVal.Launch.Diagnostics = launch.GetDiagnostics()
		resVal.Launch.Init = launch.GetInit()
		resVal.Launch.Validate = launch.GetValidate()
		resVal.Launch.Plan = launch.GetPlan()
		resVal.Launch.Apply = launch.GetApply()

		// Determine our final status from all operations
		res.Status = diagnostics.OperationStatus(runner.TFConfig.FailOnWarnings, res)

		return res
	}
}

// scenarioLaunch initializes, validates, plans, and applies the generated Terraform module
func (e *Runner) scenarioLaunch(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
) *pb.Operation_Response_Launch_ {
	// Create the check response value
	res := &pb.Operation_Response_Launch_{
		Launch: &pb.Operation_Response_Launch{},
	}

	// check the Terraform module
	checkRes := e.scenarioCheck(ctx, req, events)

	res.Launch.Diagnostics = checkRes.Check.GetDiagnostics()
	res.Launch.Init = checkRes.Check.GetInit()
	res.Launch.Validate = checkRes.Check.GetValidate()
	res.Launch.Plan = checkRes.Check.GetPlan()

	// Return early if we failed to check our module
	if diagnostics.HasFailed(
		e.TFConfig.FailOnWarnings,
		res.Launch.Diagnostics,
		res.Launch.Init.GetDiagnostics(),
		res.Launch.Validate.GetDiagnostics(),
		res.Launch.Plan.GetDiagnostics(),
	) {
		return res
	}

	res.Launch.Apply = e.terraformApply(ctx, req, events)

	return res
}
