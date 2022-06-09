package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// writeDecodeResponse takes a scenario decode response and writes human
// readable output.
func (v *View) writeDecodeResponse(out *pb.Scenario_Operation_Decode_Response) bool {
	if out == nil {
		return false
	}

	if status.HasFailed(v.settings.GetFailOnWarnings(), out) {
		msg := "Decode: failed!"
		if v.settings.IsTty {
			msg = "Decode: ❌"
		}
		v.ui.Error(msg)
		v.WriteDiagnostics(out.GetDiagnostics())
		return true
	}

	var msg string
	if status.HasWarningDiags(out) {
		msg = "Decode: success! (warnings present)"
		if v.settings.IsTty {
			msg = "Decode: ⚠️"
		}
		v.ui.Warn(msg)
	} else {
		msg = "Decode: success!"
		if v.settings.IsTty {
			msg = "Decode: ✅"
		}
		v.ui.Debug(msg)
	}

	v.WriteDiagnostics(out.GetDiagnostics())

	return diagnostics.HasErrors(out.GetDiagnostics())
}

// writeGenerateResponse takes a scenario generate response and writes human
// readable output.
func (v *View) writeGenerateResponse(out *pb.Scenario_Operation_Generate_Response) bool {
	if out == nil {
		return false
	}

	if status.HasFailed(v.settings.GetFailOnWarnings(), out) {
		msg := "  Generate: failed!"
		if v.settings.IsTty {
			msg = "  Generate: ❌"
		}
		v.ui.Error(fmt.Sprintf("  Module path: %s", out.GetTerraformModule().GetModulePath()))
		v.ui.Error(fmt.Sprintf("  Module rc path: %s", out.GetTerraformModule().GetRcPath()))
		v.ui.Error(msg)
		v.WriteDiagnostics(out.GetDiagnostics())
		return true
	}

	var msg string
	if status.HasWarningDiags(out) {
		msg = "  Generate: success! (warnings present)"
		if v.settings.IsTty {
			msg = "  Generate: ⚠️"
		}
	} else {
		msg = "  Generate: success!"
		if v.settings.IsTty {
			msg = "  Generate: ✅"
		}
	}

	v.ui.Info(msg)
	v.ui.Debug(fmt.Sprintf("  Module path: %s", out.GetTerraformModule().GetModulePath()))
	v.ui.Debug(fmt.Sprintf("  Module rc path: %s", out.GetTerraformModule().GetRcPath()))
	v.WriteDiagnostics(out.GetDiagnostics())

	return diagnostics.HasErrors(out.GetDiagnostics())
}

func (v *View) generateResponseWriter(out *pb.Scenario_Operation_Generate_Response) func() bool {
	return func() bool {
		return v.writeGenerateResponse(out)
	}
}
