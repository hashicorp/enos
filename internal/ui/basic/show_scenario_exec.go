package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenarioExec shows scenario exec view
func (v *View) ShowScenarioExec(res *pb.ExecScenariosResponse) error {
	for _, out := range res.GetResponses() {
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())

		v.ui.Info(fmt.Sprintf("Scenario: %s", scenario.String()))
		_ = v.writeExecResponse(out.GetSubCommand(), out.GetExec())
	}

	v.WriteDiagnostics(res.GetDiagnostics())

	return status.ExecScenarios(res)
}
