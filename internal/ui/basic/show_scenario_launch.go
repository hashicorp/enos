package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenarioLaunch shows scenario launch view
func (v *View) ShowScenarioLaunch(res *pb.LaunchScenariosResponse) error {
	v.writeDecodeResponse(res.GetDecode())

	for _, out := range res.GetResponses() {
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetGenerate().GetTerraformModule().GetScenarioRef())

		v.ui.Info(fmt.Sprintf("Scenario: %s", scenario.String()))
		v.writeUntilFailure([]func() bool{
			v.generateResponseWriter(out.GetGenerate()),
			v.initResponseWriter(out.GetInit()),
			v.validateResponseWriter(out.GetValidate()),
			v.planResponseWriter(out.GetPlan()),
			v.applyResponseWriter(out.GetApply()),
		})
	}

	v.WriteDiagnostics(res.GetDiagnostics())

	return status.LaunchScenarios(v.settings.GetFailOnWarnings(), res)
}
