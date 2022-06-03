package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenarioValidate shows scenario generate view
func (v *View) ShowScenarioValidate(res *pb.ValidateScenariosResponse) error {
	for _, out := range res.GetResponses() {
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetGenerate().GetTerraformModule().GetScenarioRef())

		v.ui.Info(fmt.Sprintf("Scenario: %s", scenario.String()))
		v.writeUntilFailure([]func() bool{
			v.generateResponseWriter(out.GetGenerate()),
			v.initResponseWriter(out.GetInit()),
			v.validateResponseWriter(out.GetValidate()),
			v.planResponseWriter(out.GetPlan()),
		})
	}

	v.WriteDiagnostics(res.GetDiagnostics())

	return status.ValidateScenarios(v.settings.GetFailOnWarnings(), res)
}
