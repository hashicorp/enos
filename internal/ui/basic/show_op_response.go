// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui/status"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

var failedIcons = []string{"🙈", "🙉", "🙊"}

// ShowOperationResponses shows an operation responses.
func (v *View) ShowOperationResponses(res *pb.OperationResponses) error {
	// Determine if anything failed so we can create the correct display header
	diags := diagnostics.Concat(res.GetDecode().GetDiagnostics(), res.GetDiagnostics())
	failed := diagnostics.HasFailed(v.Settings().GetFailOnWarnings(), diags)
	if !failed {
		for _, r := range res.GetResponses() {
			if diagnostics.OpResFailed(v.Settings().GetFailOnWarnings(), r) {
				failed = true

				break
			}
		}
	}

	header := "\nEnos operations"
	if failed {
		if v.settings.GetIsTty() {
			//nolint:gosec // G404 it's okay to use weak random numbers for random failed icons
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			header = fmt.Sprintf("%s failed! %s\n", header, failedIcons[r.Intn(len(failedIcons))])
		} else {
			header += " failed!\n"
		}
	} else {
		if v.settings.GetIsTty() {
			header += " finished! 🐵\n"
		} else {
			header += " finished!\n"
		}
	}

	if diagnostics.HasFailed(v.Settings().GetFailOnWarnings(), diags) {
		// Our request failed so show our header and request diagnostics
		v.ui.Error(header)

		err := v.ShowDiagnostics(diags)
		if err != nil {
			return err
		}

		return status.OperationResponses(v.Settings().GetFailOnWarnings(), res)
	}

	if failed {
		// One or more sub-requests failed
		v.ui.Error(header)
	} else {
		v.ui.Info(header)
	}

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

// ShowOperationResponse shows an operation response.
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

// showOperationResponse shows an operation response.
func (v *View) showOperationResponse(res *pb.Operation_Response, fullOnComplete bool) error {
	if res == nil {
		return nil
	}

	scenario := flightplan.NewScenario()
	scenario.FromRef(res.GetOp().GetScenario())
	v.ui.Info(fmt.Sprintf("Scenario: %s %s", scenario.String(), v.opStatusString(res.GetStatus())))

	if !shouldShowFullResponse(res, fullOnComplete) {
		return nil
	}

	switch t := res.GetValue().(type) {
	case *pb.Operation_Response_Generate_:
		v.writeGenerateResponse(res.GetGenerate())
	case *pb.Operation_Response_Check_:
		v.writeGenerateResponse(res.GetGenerate())
		v.writeInitResponse(res.GetCheck().GetInit())
		v.writeValidateResponse(res.GetCheck().GetValidate())
		if plan := res.GetCheck().GetPlan(); plan != nil {
			v.writePlainTextResponse("plan", plan.GetStderr(), plan)
		}
	case *pb.Operation_Response_Launch_:
		v.writeGenerateResponse(res.GetGenerate())
		v.writeInitResponse(res.GetLaunch().GetInit())
		v.writeValidateResponse(res.GetLaunch().GetValidate())
		if plan := res.GetLaunch().GetPlan(); plan != nil {
			v.writePlainTextResponse("plan", plan.GetStderr(), plan)
		}
		if apply := res.GetLaunch().GetApply(); apply != nil {
			v.writePlainTextResponse("apply", apply.GetStderr(), apply)
		}
	case *pb.Operation_Response_Destroy_:
		v.writeGenerateResponse(res.GetGenerate())
		v.writeInitResponse(res.GetDestroy().GetInit())
		if show := res.GetDestroy().GetPriorStateShow(); show != nil {
			v.writeShowResponse(show)
		}
		if destroy := res.GetDestroy().GetDestroy(); destroy != nil {
			v.writePlainTextResponse("destroy", destroy.GetStderr(), destroy)
		}
	case *pb.Operation_Response_Run_:
		v.writeGenerateResponse(res.GetGenerate())
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

// shouldShowFullResponse determines whether or not we should display the entire
// operation response. We show the full response if something went wrong, if
// a full response has been requested, or if there is operation information for
// sub-operations that have output like exec or output.
func shouldShowFullResponse(res *pb.Operation_Response, fullOnComplete bool) bool {
	if fullOnComplete {
		return true
	}

	if res.GetStatus() != pb.Operation_STATUS_COMPLETED {
		return true
	}

	if res.GetExec() != nil {
		return true
	}

	if res.GetOutput() != nil {
		return true
	}

	return false
}
