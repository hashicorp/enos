// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"github.com/hashicorp/enos/internal/ui/status"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// writeDecodeResponse takes a scenario decode response and writes human
// readable output.
func (v *View) writeDecodeResponse(out *pb.DecodeResponse) {
	if out == nil {
		return
	}

	if status.HasFailed(v.settings.GetFailOnWarnings(), out) {
		msg := "Decode: failed!"
		if v.settings.GetIsTty() {
			msg = "Decode: ❌"
		}
		v.ui.Error(msg)
		v.WriteDiagnostics(out.GetDiagnostics())

		return
	}

	var msg string
	if status.HasWarningDiags(out) {
		msg = "Decode: success! (warnings present)"
		if v.settings.GetIsTty() {
			msg = "Decode: ⚠️"
		}
		v.ui.Warn(msg)
	} else {
		msg = "Decode: success!"
		if v.settings.GetIsTty() {
			msg = "Decode: ✅"
		}
		v.ui.Debug(msg)
	}

	v.WriteDiagnostics(out.GetDiagnostics())
}

// writeGenerateResponse takes a scenario generate response and writes human
// readable output.
func (v *View) writeGenerateResponse(out *pb.Operation_Response_Generate) {
	if out == nil {
		return
	}

	if status.HasFailed(v.settings.GetFailOnWarnings(), out) {
		msg := "  Generate: failed!"
		if v.settings.GetIsTty() {
			msg = "  Generate: ❌"
		}
		v.ui.Error("  Module path: " + out.GetTerraformModule().GetModulePath())
		v.ui.Error("  Module rc path: " + out.GetTerraformModule().GetRcPath())
		v.ui.Error(msg)
		v.WriteDiagnostics(out.GetDiagnostics())

		return
	}

	var msg string
	if status.HasWarningDiags(out) {
		msg = "  Generate: success! (warnings present)"
		if v.settings.GetIsTty() {
			msg = "  Generate: ⚠️"
		}
	} else {
		msg = "  Generate: success!"
		if v.settings.GetIsTty() {
			msg = "  Generate: ✅"
		}
	}

	v.ui.Info(msg)
	v.ui.Debug("  Module path: " + out.GetTerraformModule().GetModulePath())
	v.ui.Debug("  Module rc path: " + out.GetTerraformModule().GetRcPath())
	v.WriteDiagnostics(out.GetDiagnostics())
}
