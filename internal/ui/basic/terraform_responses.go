package basic

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/execute/terraform/format"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func (v *View) writeInitResponse(init *pb.Terraform_Command_Init_Response) {
	if init == nil {
		return
	}

	v.writePlainTextResponse("init", init.GetStderr(), init.GetDiagnostics())
}

func (v *View) writeValidateResponse(validate *pb.Terraform_Command_Validate_Response) {
	if validate == nil {
		return
	}

	if len(validate.GetDiagnostics()) > 0 || !validate.GetValid() {
		msg := "  Validate: failed!"
		if v.settings.IsTty {
			msg = "  Validate: ❌"
		}
		v.ui.Error(msg)
		v.ui.Error(fmt.Sprintf("  Validation errors: %d", validate.GetErrorCount()))
		v.ui.Error(fmt.Sprintf("  Validation warnings: %d", validate.GetWarningCount()))
		v.ui.Debug(fmt.Sprintf("  Validation format: %s", validate.GetFormatVersion()))
		v.WriteDiagnostics(validate.GetDiagnostics())
		return
	}

	msg := "  Validate: success!"
	if v.settings.IsTty {
		msg = "  Validate: ✅"
	}
	v.ui.Info(msg)
	v.ui.Debug(fmt.Sprintf("  Validation errors: %d", validate.GetErrorCount()))
	v.ui.Debug(fmt.Sprintf("  Validation warnings: %d", validate.GetWarningCount()))
	v.ui.Debug(fmt.Sprintf("  Validation format: %s", validate.GetFormatVersion()))
}

func (v *View) writePlanResponse(plan *pb.Terraform_Command_Plan_Response) {
	if plan == nil {
		return
	}

	v.writePlainTextResponse("plan", plan.GetStderr(), plan.GetDiagnostics())
}

func (v *View) writeApplyResponse(apply *pb.Terraform_Command_Apply_Response) {
	if apply == nil {
		return
	}

	v.writePlainTextResponse("apply", apply.GetStderr(), apply.GetDiagnostics())
}

func (v *View) writeDestroyResponse(destroy *pb.Terraform_Command_Destroy_Response) {
	if destroy == nil {
		return
	}

	v.writePlainTextResponse("destroy", destroy.GetStderr(), destroy.GetDiagnostics())
}

func (v *View) writeExecResponse(subCmd string, exec *pb.Terraform_Command_Exec_Response) {
	if exec == nil {
		return
	}

	if len(exec.GetDiagnostics()) > 0 {
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
		return
	}

	v.ui.Info(exec.GetStdout())
	v.ui.Debug(fmt.Sprintf("  Sub-command: %s", subCmd))
}

func (v *View) writeOutputResponse(out *pb.Terraform_Command_Output_Response) {
	if out == nil {
		return
	}

	if len(out.GetDiagnostics()) > 0 {
		msg := "  Output: failed!"
		if v.settings.IsTty {
			msg = "  Output: ❌"
		}
		v.ui.Error(msg)
		v.WriteDiagnostics(out.GetDiagnostics())
		return
	}

	for _, meta := range out.GetMeta() {
		s, err := format.TerraformOutput(meta, 2)
		if err != nil {
			v.WriteDiagnostics(diagnostics.FromErr(err))
		} else {
			v.ui.Info(fmt.Sprintf("  %s = %s", meta.GetName(), s))
		}
	}
}

func (v *View) writePlainTextResponse(cmd string, stderr string, diagnotsics []*pb.Diagnostic) {
	cmd = cases.Title(language.English).String(cmd)
	if len(diagnotsics) > 0 {
		msg := fmt.Sprintf("  %s: failed!", cmd)
		if v.settings.IsTty {
			msg = fmt.Sprintf(" %s: ❌", cmd)
		}
		v.ui.Error(msg)
		if stderr != "" {
			v.ui.Error(stderr)
		}
		v.WriteDiagnostics(diagnotsics)
		return
	}

	msg := fmt.Sprintf("  %s: success!", cmd)
	if v.settings.IsTty {
		msg = fmt.Sprintf("  %s: ✅", cmd)
	}
	v.ui.Info(msg)
}
