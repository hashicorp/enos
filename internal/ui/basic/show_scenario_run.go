package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenarioRun shows scenario run view
func (v *View) ShowScenarioRun(res *pb.RunScenariosResponse) error {
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
			v.destroyResponseWriter(out.GetDestroy()),
		})
	}

	v.WriteDiagnostics(res.GetDecode().GetDiagnostics())
	v.WriteDiagnostics(res.GetDiagnostics())

	return status.RunScenarios(v.settings.GetFailOnWarnings(), res)
}
