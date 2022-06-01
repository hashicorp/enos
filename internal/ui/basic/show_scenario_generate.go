package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func (v *View) writeGenerateResponse(out *pb.Scenario_Command_Generate_Response) {
	if out == nil {
		return
	}

	if len(out.GetDiagnostics()) > 0 {
		msg := "  Generate: failed!"
		if v.settings.IsTty {
			msg = "  Generate: ❌"
		}
		v.ui.Error(fmt.Sprintf("  Module path: %s", out.GetTerraformModule().GetModulePath()))
		v.ui.Error(fmt.Sprintf("  Module rc path: %s", out.GetTerraformModule().GetRcPath()))
		v.ui.Error(msg)
		v.WriteDiagnostics(out.GetDiagnostics())
		return
	}

	msg := "  Generate: success!"
	if v.settings.IsTty {
		msg = "  Generate: ✅"
	}
	v.ui.Info(msg)
	v.ui.Debug(fmt.Sprintf("  Module path: %s", out.GetTerraformModule().GetModulePath()))
	v.ui.Debug(fmt.Sprintf("  Module rc path: %s", out.GetTerraformModule().GetRcPath()))
	v.WriteDiagnostics(out.GetDiagnostics())
}

// ShowScenarioGenerate shows scenario generate view
func (v *View) ShowScenarioGenerate(res *pb.GenerateScenariosResponse) error {
	for _, out := range res.GetResponses() {
		v.writeGenerateResponse(out)
	}

	v.WriteDiagnostics(res.GetDiagnostics())

	return status.GenerateScenarios(res)
}
