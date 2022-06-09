package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenarioDestroy shows scenario destroy view
func (v *View) ShowScenarioDestroy(res *pb.DestroyScenariosResponse) error {
	v.writeDecodeResponse(res.GetDecode())

	for _, out := range res.GetResponses() {
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())

		v.ui.Info(fmt.Sprintf("Scenario: %s", scenario.String()))
		v.writeUntilFailure([]func() bool{
			v.destroyResponseWriter(out.GetDestroy()),
		})
	}

	v.WriteDiagnostics(res.GetDiagnostics())

	return status.DestroyScenarios(v.settings.GetFailOnWarnings(), res)
}
