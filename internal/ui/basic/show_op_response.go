package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowOperationResponses shows an operation responses
func (v *View) ShowOperationResponses(res *pb.OperationResponses) error {
	// Check for request level diagnostics and fail early if they exist
	diags := diagnostics.Concat(res.GetDecode().GetDiagnostics(), res.GetDiagnostics())
	if diagnostics.HasFailed(v.Settings().GetFailOnWarnings(), diags) {
		err := v.ShowDiagnostics(diags)
		if err != nil {
			return err
		}

		return status.OperationResponses(v.Settings().GetFailOnWarnings(), res)
	}

	// Our request didn't fail so we'll show each operations status
	v.ui.Info("\nEnos operations finished!\n")

	err := v.ShowDecode(res.GetDecode(), true)
	if err != nil {
		return err
	}

	err = v.ShowDiagnostics(res.GetDiagnostics())
	if err != nil {
		return err
	}

	for _, r := range res.GetResponses() {
		err = v.showOperationResponse(r, false)
		if err != nil {
			return err
		}
	}

	return status.OperationResponses(v.Settings().GetFailOnWarnings(), res)
}

// ShowOperationResponse shows an operation response
func (v *View) ShowOperationResponse(res *pb.Operation_Response) error {
	if res == nil {
		return nil
	}

	if err := v.showOperationResponse(res, true); err != nil {
		return err
	}

	return status.OperationResponses(v.Settings().GetFailOnWarnings(), &pb.OperationResponses{
		Responses: []*pb.Operation_Response{res},
	})
}

// showOperationResponse shows an operation response
func (v *View) showOperationResponse(res *pb.Operation_Response, fullOnComplete bool) error {
	if res == nil {
		return nil
	}

	scenario := flightplan.NewScenario()
	scenario.FromRef(res.GetOp().GetScenario())
	v.ui.Info(fmt.Sprintf("Scenario: %s %s", scenario.String(), v.opStatusString(res.GetStatus())))

	// If we have a successful operation and fullOnComplete has not been set to
	// true we'll move on after printing a single line for the scenario.
	if res.GetStatus() == pb.Operation_STATUS_COMPLETED && !fullOnComplete {
		return nil
	}

	switch t := res.GetValue().(type) {
	case *pb.Operation_Response_Generate_:
		v.writeGenerateResponse(res.GetGenerate())
	case *pb.Operation_Response_Check_:
		v.writeInitResponse(res.GetCheck().GetInit())
		v.writeValidateResponse(res.GetCheck().GetValidate())
		if plan := res.GetCheck().GetPlan(); plan != nil {
			v.writePlainTextResponse("plan", plan.GetStderr(), plan)
		}
	case *pb.Operation_Response_Launch_:
		v.writeInitResponse(res.GetLaunch().GetInit())
		v.writeValidateResponse(res.GetLaunch().GetValidate())
		if plan := res.GetLaunch().GetPlan(); plan != nil {
			v.writePlainTextResponse("plan", plan.GetStderr(), plan)
		}
		if apply := res.GetLaunch().GetApply(); apply != nil {
			v.writePlainTextResponse("apply", apply.GetStderr(), apply)
		}
	case *pb.Operation_Response_Destroy_:
		v.writeInitResponse(res.GetDestroy().GetInit())
		if show := res.GetDestroy().GetPriorStateShow(); show != nil {
			v.writeShowResponse(show)
		}
		if destroy := res.GetDestroy().GetDestroy(); destroy != nil {
			v.writePlainTextResponse("destroy", destroy.GetStderr(), destroy)
		}
	case *pb.Operation_Response_Run_:
		v.writeInitResponse(res.GetRun().GetInit())
		v.writeValidateResponse(res.GetRun().GetValidate())
		if plan := res.GetRun().GetPlan(); plan != nil {
			v.writePlainTextResponse("plan", plan.GetStderr(), plan)
		}
		if apply := res.GetRun().GetApply(); apply != nil {
			v.writePlainTextResponse("apply", apply.GetStderr(), apply)
		}
		if show := res.GetRun().GetPriorStateShow(); show != nil {
			v.writeShowResponse(show)
		}
		if destroy := res.GetRun().GetDestroy(); destroy != nil {
			v.writePlainTextResponse("destroy", destroy.GetStderr(), destroy)
		}
	case *pb.Operation_Response_Exec_:
		v.writeExecResponse(res.GetExec().GetExec())
	case *pb.Operation_Response_Output_:
		v.writeOutputResponse(res)
	default:
		return fmt.Errorf("unable to display operation response value '%t:%[1]v'", t)
	}

	return nil
}
