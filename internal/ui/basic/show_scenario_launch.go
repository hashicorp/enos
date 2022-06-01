package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/server/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenarioLaunch shows scenario launch view
func (v *View) ShowScenarioLaunch(res *pb.LaunchScenariosResponse) error {
	for _, out := range res.GetResponses() {
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetGenerate().GetTerraformModule().GetScenarioRef())

		v.ui.Info(fmt.Sprintf("Scenario: %s", scenario.String()))
		v.writeGenerateResponse(out.GetGenerate())
		v.writeInitResponse(out.GetInit())
		v.writeValidateResponse(out.GetValidate())
		v.writePlanResponse(out.GetPlan())
		v.writeApplyResponse(out.GetApply())
	}

	v.WriteDiagnostics(res.GetDiagnostics())

	return status.LaunchScenarios(res)
}
