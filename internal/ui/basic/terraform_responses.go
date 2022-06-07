package basic

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/execute/terraform/format"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func (v *View) writeInitResponse(init *pb.Terraform_Command_Init_Response) bool {
	if init == nil {
		return true
	}

	return v.writePlainTextResponse("init", init.GetStderr(), init)
}

func (v *View) initResponseWriter(init *pb.Terraform_Command_Init_Response) func() bool {
	return func() bool {
		return v.writeInitResponse(init)
	}
}

func (v *View) writeValidateResponse(validate *pb.Terraform_Command_Validate_Response) bool {
	if validate == nil {
		return false
	}

	if status.HasFailed(v.settings.FailOnWarnings, validate) {
		msg := "  Validate: failed!"
		if v.settings.IsTty {
			msg = "  Validate: ❌"
		}
		v.ui.Error(msg)
		v.ui.Error(fmt.Sprintf("  Validation errors: %d", validate.GetErrorCount()))
		v.ui.Error(fmt.Sprintf("  Validation warnings: %d", validate.GetWarningCount()))
		v.ui.Debug(fmt.Sprintf("  Validation format: %s", validate.GetFormatVersion()))
		v.WriteDiagnostics(validate.GetDiagnostics())

		return true
	}

	var msg string
	if status.HasWarningDiags(validate) {
		msg = "  Validate: success! (warnings present)"
		if v.settings.IsTty {
			msg = "  Validate: ⚠️"
		}
	} else {
		msg = "  Validate: success!"
		if v.settings.IsTty {
			msg = "  Validate: ✅"
		}
	}
	v.ui.Info(msg)
	v.ui.Debug(fmt.Sprintf("  Validation errors: %d", validate.GetErrorCount()))
	v.ui.Debug(fmt.Sprintf("  Validation warnings: %d", validate.GetWarningCount()))
	v.ui.Debug(fmt.Sprintf("  Validation format: %s", validate.GetFormatVersion()))

	return false
}

func (v *View) validateResponseWriter(validate *pb.Terraform_Command_Validate_Response) func() bool {
	return func() bool {
		return v.writeValidateResponse(validate)
	}
}

func (v *View) writePlanResponse(plan *pb.Terraform_Command_Plan_Response) bool {
	if plan == nil {
		return false
	}

	return v.writePlainTextResponse("plan", plan.GetStderr(), plan)
}

func (v *View) planResponseWriter(plan *pb.Terraform_Command_Plan_Response) func() bool {
	return func() bool {
		return v.writePlanResponse(plan)
	}
}

func (v *View) writeApplyResponse(apply *pb.Terraform_Command_Apply_Response) bool {
	if apply == nil {
		return false
	}

	return v.writePlainTextResponse("apply", apply.GetStderr(), apply)
}

func (v *View) applyResponseWriter(apply *pb.Terraform_Command_Apply_Response) func() bool {
	return func() bool {
		return v.writeApplyResponse(apply)
	}
}

func (v *View) writeDestroyResponse(destroy *pb.Terraform_Command_Destroy_Response) bool {
	if destroy == nil {
		return false
	}

	return v.writePlainTextResponse("destroy", destroy.GetStderr(), destroy)
}

func (v *View) destroyResponseWriter(destroy *pb.Terraform_Command_Destroy_Response) func() bool {
	return func() bool {
		return v.writeDestroyResponse(destroy)
	}
}

func (v *View) writeExecResponse(subCmd string, exec *pb.Terraform_Command_Exec_Response) bool {
	if exec == nil {
		return false
	}

	if status.HasFailed(v.settings.FailOnWarnings, exec) {
		msg := "  Exec: failed!"
		if v.settings.IsTty {
			msg = "  Exec: ❌"
		}
		v.ui.Error(fmt.Sprintf("  Sub-command: %s", subCmd))
		v.ui.Error(msg)
		if stdout := exec.GetStdout(); stdout != "" {
			v.ui.Error(stdout)
		}
		if stderr := exec.GetStderr(); stderr != "" {
			v.ui.Error(stderr)
		}
		v.WriteDiagnostics(exec.GetDiagnostics())

		return true
	}

	v.ui.Info(exec.GetStdout())
	v.ui.Debug(fmt.Sprintf("  Sub-command: %s", subCmd))
	v.WriteDiagnostics(exec.GetDiagnostics())

	return false
}

func (v *View) execResponseWriter(subCmd string, exec *pb.Terraform_Command_Exec_Response) func() bool {
	return func() bool {
		return v.writeExecResponse(subCmd, exec)
	}
}

func (v *View) writeOutputResponse(out *pb.Terraform_Command_Output_Response) bool {
	if out == nil {
		return false
	}

	if status.HasFailed(v.settings.FailOnWarnings, out) {
		msg := "  Output: failed!"
		if v.settings.IsTty {
			msg = "  Output: ❌"
		}
		v.ui.Error(msg)
		v.WriteDiagnostics(out.GetDiagnostics())

		return true
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

	return false
}

func (v *View) outputResponseWriter(out *pb.Terraform_Command_Output_Response) func() bool {
	return func() bool {
		return v.writeOutputResponse(out)
	}
}

func (v *View) writePlainTextResponse(cmd string, stderr string, res status.ResWithDiags) bool {
	cmd = cases.Title(language.English).String(cmd)
	if status.HasFailed(v.settings.FailOnWarnings, res) {
		msg := fmt.Sprintf("  %s: failed!", cmd)
		if v.settings.IsTty {
			msg = fmt.Sprintf(" %s: ❌", cmd)
		}
		v.ui.Error(msg)
		if stderr != "" {
			v.ui.Error(stderr)
		}
		v.WriteDiagnostics(res.GetDiagnostics())
		return true
	}

	var msg string
	if status.HasWarningDiags(res) {
		msg = fmt.Sprintf("  %s: success! (warnings present)", cmd)
		if v.settings.IsTty {
			msg = fmt.Sprintf("  %s: ⚠️", cmd)
		}
	} else {
		msg = fmt.Sprintf("  %s: success!", cmd)
		if v.settings.IsTty {
			msg = fmt.Sprintf("  %s: ✅", cmd)
		}
	}
	v.ui.Info(msg)
	return false
}

func (v *View) writeUntilFailure(fs []func() bool) {
	for _, f := range fs {
		if failed := f(); failed {
			return
		}
	}
}
