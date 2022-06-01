package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/server/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenarioRun shows scenario run view
func (v *View) ShowScenarioRun(res *pb.RunScenariosResponse) error {
	for _, out := range res.GetResponses() {
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetGenerate().GetTerraformModule().GetScenarioRef())

		v.ui.Info(fmt.Sprintf("Scenario: %s", scenario.String()))
		v.writeGenerateResponse(out.GetGenerate())
		v.writeInitResponse(out.GetInit())
		v.writeValidateResponse(out.GetValidate())
		v.writePlanResponse(out.GetPlan())
		v.writeApplyResponse(out.GetApply())
		v.writeDestroyResponse(out.GetDestroy())
	}

	v.WriteDiagnostics(res.GetDiagnostics())

	return status.RunScenarios(res)
}
