package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenarioGenerate shows scenario generate view
func (v *View) ShowScenarioGenerate(res *pb.GenerateScenariosResponse) error {
	v.writeDecodeResponse(res.GetDecode())

	for _, out := range res.GetResponses() {
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())

		v.ui.Info(fmt.Sprintf("Scenario: %s", scenario.String()))
		v.writeUntilFailure([]func() bool{
			v.generateResponseWriter(out),
		})
	}

	v.WriteDiagnostics(res.GetDiagnostics())

	return status.GenerateScenarios(v.settings.GetFailOnWarnings(), res)
}
