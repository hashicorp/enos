// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/operation/terraform/format"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func (v *View) writeInitResponse(init *pb.Terraform_Command_Init_Response) {
	if init == nil {
		return
	}

	v.writePlainTextResponse("init", init.GetStderr(), init)
}

func (v *View) writeValidateResponse(validate *pb.Terraform_Command_Validate_Response) {
	if validate == nil {
		return
	}

	if status.HasFailed(v.settings.GetFailOnWarnings(), validate) {
		msg := "  Validate: failed!"
		if v.settings.GetIsTty() {
			msg = "  Validate: ❌"
		}
		v.ui.Error(msg)
		v.ui.Error(fmt.Sprintf("  Validation errors: %d", validate.GetErrorCount()))
		v.ui.Error(fmt.Sprintf("  Validation warnings: %d", validate.GetWarningCount()))
		v.ui.Debug("  Validation format: " + validate.GetFormatVersion())
		v.WriteDiagnostics(validate.GetDiagnostics())

		return
	}

	var msg string
	if status.HasWarningDiags(validate) {
		msg = "  Validate: success! (warnings present)"
		if v.settings.GetIsTty() {
			msg = "  Validate: ⚠️"
		}
	} else {
		msg = "  Validate: success!"
		if v.settings.GetIsTty() {
			msg = "  Validate: ✅"
		}
	}
	v.ui.Info(msg)
	v.ui.Debug(fmt.Sprintf("  Validation errors: %d", validate.GetErrorCount()))
	v.ui.Debug(fmt.Sprintf("  Validation warnings: %d", validate.GetWarningCount()))
	v.ui.Debug("  Validation format: " + validate.GetFormatVersion())
	v.WriteDiagnostics(validate.GetDiagnostics())
}

func (v *View) writeExecResponse(exec *pb.Terraform_Command_Exec_Response) {
	if exec == nil {
		return
	}

	if status.HasFailed(v.settings.GetFailOnWarnings(), exec) {
		msg := "  Exec: failed!"
		if v.settings.GetIsTty() {
			msg = "  Exec: ❌"
		}
		v.ui.Error("  Sub-command: " + exec.GetSubCommand())
		v.ui.Error(msg)
		if stdout := exec.GetStdout(); stdout != "" {
			v.ui.Error(stdout)
		}
		if stderr := exec.GetStderr(); stderr != "" {
			v.ui.Error(stderr)
		}
		v.WriteDiagnostics(exec.GetDiagnostics())

		return
	}

	v.ui.Info(exec.GetStdout())
	v.ui.Error(exec.GetStderr())
	v.ui.Debug("  Sub-command: " + exec.GetSubCommand())
	v.WriteDiagnostics(exec.GetDiagnostics())
}

func (v *View) writeOutputResponse(res *pb.Operation_Response) {
	if res == nil {
		return
	}

	out := res.GetOutput().GetOutput()
	if out == nil {
		return
	}

	scenario := flightplan.NewScenario()
	scenario.FromRef(res.GetOp().GetScenario())
	v.ui.Info(fmt.Sprintf("Scenario: %s %s", scenario.String(), v.opStatusString(res.GetStatus())))

	if status.HasFailed(v.settings.GetFailOnWarnings(), out) {
		msg := "  Output: failed!"
		if v.settings.GetIsTty() {
			msg = "  Output: ❌"
		}
		v.ui.Error(msg)
		v.WriteDiagnostics(out.GetDiagnostics())
	}

	diags := out.GetDiagnostics()
	for _, meta := range out.GetMeta() {
		s, err := format.TerraformOutput(meta, 2)
		if err != nil {
			diags = append(diags, diagnostics.FromErr(err)...)
		} else {
			v.ui.Info(fmt.Sprintf("  %s = %s", meta.GetName(), s))
		}
	}
	v.WriteDiagnostics(diags)
}

func (v *View) writeShowResponse(show *pb.Terraform_Command_Show_Response) {
	if show == nil {
		return
	}

	if status.HasFailed(v.settings.GetFailOnWarnings(), show) {
		msg := "  Read state: failed!"
		if v.settings.GetIsTty() {
			msg = "  Read state: ❌"
		}

		if s := string(show.GetState()); s != "" {
			v.ui.Error("  State: " + s)
			v.ui.Error(msg)
		}

		v.WriteDiagnostics(show.GetDiagnostics())

		return
	}

	var msg string
	if status.HasWarningDiags(show) {
		msg = "  Read state: success! (warnings present)"
		if v.settings.GetIsTty() {
			msg = "  Read state: ⚠️"
		}
	} else {
		msg = "  Read state: success!"
		if v.settings.GetIsTty() {
			msg = "  Read state: ✅"
		}
	}
	v.ui.Info(msg)
	if s := string(show.GetState()); s != "" {
		v.ui.Debug("  State: " + s)
	}

	v.WriteDiagnostics(show.GetDiagnostics())
}

func (v *View) writePlainTextResponse(cmd string, stderr string, res status.ResWithDiags) {
	if cmd == "" {
		return
	}

	cmd = cases.Title(language.English).String(cmd)
	if status.HasFailed(v.settings.GetFailOnWarnings(), res) {
		msg := fmt.Sprintf("  %s: failed!", cmd)
		if v.settings.GetIsTty() {
			msg = fmt.Sprintf("  %s: ❌", cmd)
		}
		v.ui.Error(msg)
		if stderr != "" {
			v.ui.Error(stderr)
		}
		v.WriteDiagnostics(res.GetDiagnostics())

		return
	}

	var msg string
	if status.HasWarningDiags(res) {
		msg = fmt.Sprintf("  %s: success! (warnings present)", cmd)
		if v.settings.GetIsTty() {
			msg = fmt.Sprintf("  %s: ⚠️", cmd)
		}
	} else {
		msg = fmt.Sprintf("  %s: success!", cmd)
		if v.settings.GetIsTty() {
			msg = fmt.Sprintf("  %s: ✅", cmd)
		}
	}
	v.ui.Info(msg)
	v.WriteDiagnostics(res.GetDiagnostics())
}
