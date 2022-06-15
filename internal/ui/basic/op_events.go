package basic

import (
	"fmt"
	"strings"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/operation/terraform/format"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func (v *View) writeEventDecode(e *pb.Operation_Event, w *strings.Builder) {
	v.writeEventHeader("decode", e, pb.UI_Settings_LEVEL_DEBUG, w)

	// Write debug information if necessary
	if v.settings.Level > pb.UI_Settings_LEVEL_INFO {
		if fp := e.GetDecode().GetFlightplan(); fp != nil {
			extra := strings.Builder{}

			if bd := fp.GetBaseDir(); bd != "" {
				extra.WriteString(fmt.Sprintf("  Base directory: %s\n", bd))
			}
			hclFiles := fp.GetEnosHcl()
			if len(hclFiles) > 0 {
				extra.WriteString("   With files:\n")
				for path := range hclFiles {
					extra.WriteString(fmt.Sprintf("     %s\n", path))
				}
			}
			varsFiles := fp.GetEnosVarsHcl()
			if len(varsFiles) > 0 {
				extra.WriteString("   With variable files:\n")
				for path := range varsFiles {
					extra.WriteString(fmt.Sprintf("     %s\n", path))
				}
			}
			envVars := fp.GetEnosVarsEnv()
			if len(envVars) > 0 {
				extra.WriteString("   With environment variables:\n")
				for _, env := range envVars {
					extra.WriteString(fmt.Sprintf("     %s\n", env))
				}
			}

			if xi := extra.String(); xi != "" {
				w.WriteString(fmt.Sprintf("\n%s", strings.TrimRight(xi, "\n")))
			}
		}
	}

	v.writeDiags(e.GetDecode().GetDiagnostics(), w)
}

func (v *View) writeEventGenerate(e *pb.Operation_Event, w *strings.Builder) {
	v.writeEventHeader("generate", e, pb.UI_Settings_LEVEL_INFO, w)

	g := e.GetGenerate()
	if g == nil {
		return
	}

	if v.settings.Level > pb.UI_Settings_LEVEL_INFO {
		extra := strings.Builder{}

		if mp := g.GetTerraformModule().GetModulePath(); mp != "" {
			extra.WriteString(fmt.Sprintf("  Module path: %s\n", mp))
		}
		if rcp := g.GetTerraformModule().GetRcPath(); rcp != "" {
			extra.WriteString(fmt.Sprintf("  Module rc path: %s\n", rcp))
		}

		if xi := extra.String(); xi != "" {
			w.WriteString(fmt.Sprintf("\n%s", strings.TrimRight(xi, "\n")))
		}
	}

	v.writeDiags(g.GetDiagnostics(), w)
}

func (v *View) writeEventInit(e *pb.Operation_Event, w *strings.Builder) {
	i := e.GetInit()
	if i == nil {
		return
	}

	v.writeEventHeader("init", e, pb.UI_Settings_LEVEL_INFO, w)

	if stderr := i.GetStderr(); stderr != "" &&
		v.settings.Level > pb.UI_Settings_LEVEL_INFO {
		w.WriteString(fmt.Sprintf("\n  Stderr: %s\n", stderr))
	}

	v.writeDiags(i.GetDiagnostics(), w)
}

func (v *View) writeEventValidate(e *pb.Operation_Event, w *strings.Builder) {
	v.writeEventHeader("validate", e, pb.UI_Settings_LEVEL_INFO, w)

	vl := e.GetValidate()
	if vl == nil {
		return
	}

	extra := strings.Builder{}
	if ec := vl.GetErrorCount(); ec > 0 {
		extra.WriteString(fmt.Sprintf("  Validation errors: %d\n", ec))
	}

	if v.settings.Level >= pb.UI_Settings_LEVEL_WARN {
		if wc := vl.GetWarningCount(); wc > 0 {
			extra.WriteString(fmt.Sprintf("  Validation warnings: %d\n", wc))
		}
	}

	if v.settings.Level >= pb.UI_Settings_LEVEL_DEBUG {
		if f := vl.GetFormatVersion(); f != "" {
			extra.WriteString(fmt.Sprintf("  Validation format: %s\n", f))
		}
	}

	if xi := extra.String(); xi != "" {
		w.WriteString(fmt.Sprintf("\n%s", strings.TrimRight(xi, "\n")))
	}

	v.writeDiags(vl.GetDiagnostics(), w)
}

func (v *View) writeEventPlan(e *pb.Operation_Event, w *strings.Builder) {
	v.writeEventHeader("plan", e, pb.UI_Settings_LEVEL_INFO, w)

	p := e.GetPlan()
	if p == nil {
		return
	}

	if stderr := p.GetStderr(); stderr != "" &&
		v.settings.Level == pb.UI_Settings_LEVEL_DEBUG {
		w.WriteString(fmt.Sprintf("\n  Stderr: %s\n", stderr))
	}

	v.writeDiags(p.GetDiagnostics(), w)
}

func (v *View) writeEventApply(e *pb.Operation_Event, w *strings.Builder) {
	v.writeEventHeader("apply", e, pb.UI_Settings_LEVEL_INFO, w)

	a := e.GetApply()
	if a == nil {
		return
	}

	if stderr := a.GetStderr(); stderr != "" &&
		v.settings.Level == pb.UI_Settings_LEVEL_DEBUG {
		w.WriteString(fmt.Sprintf("\n  Stderr: %s\n", stderr))
	}

	v.writeDiags(a.GetDiagnostics(), w)
}

func (v *View) writeEventDestroy(e *pb.Operation_Event, w *strings.Builder) {
	v.writeEventHeader("destroy", e, pb.UI_Settings_LEVEL_INFO, w)

	d := e.GetDestroy()
	if d == nil {
		return
	}

	if stderr := d.GetStderr(); stderr != "" &&
		v.settings.Level == pb.UI_Settings_LEVEL_DEBUG {
		w.WriteString(fmt.Sprintf("\n  Stderr: %s\n", stderr))
	}

	v.writeDiags(d.GetDiagnostics(), w)
}

func (v *View) writeEventExec(e *pb.Operation_Event, w *strings.Builder) {
	v.writeEventHeader("exec", e, pb.UI_Settings_LEVEL_INFO, w)

	ex := e.GetExec()
	if ex == nil {
		return
	}

	extra := strings.Builder{}
	if cmd := ex.GetSubCommand(); cmd != "" &&
		v.settings.Level == pb.UI_Settings_LEVEL_DEBUG {
		extra.WriteString(fmt.Sprintf("  Sub-command: %s\n", cmd))
	}

	if stderr := ex.GetStderr(); stderr != "" &&
		v.settings.Level == pb.UI_Settings_LEVEL_DEBUG {
		extra.WriteString(fmt.Sprintf("  Stderr: %s\n", stderr))
	}

	if stdout := ex.GetStdout(); stdout != "" &&
		v.settings.Level == pb.UI_Settings_LEVEL_DEBUG {
		extra.WriteString(fmt.Sprintf("  Stdout: %s\n", stdout))
	}

	if xi := extra.String(); xi != "" {
		w.WriteString(fmt.Sprintf("\n%s", strings.TrimRight(xi, "\n")))
	}

	v.writeDiags(ex.GetDiagnostics(), w)
}

func (v *View) writeEventOutput(e *pb.Operation_Event, w *strings.Builder) {
	out := e.GetOutput()
	if out == nil {
		return
	}

	v.writeEventHeader("output", e, pb.UI_Settings_LEVEL_DEBUG, w)

	diags := out.GetDiagnostics()
	for i, meta := range out.GetMeta() {
		s, err := format.TerraformOutput(meta, 2)
		if err != nil {
			diags = append(diags, diagnostics.FromErr(err)...)
			continue
		}

		if i != 0 {
			w.WriteString("\n")
		}
		w.WriteString(fmt.Sprintf("  %s = %s", meta.GetName(), s))
	}

	v.writeDiags(diags, w)
}

func (v *View) writeEventHeader(
	action string,
	event *pb.Operation_Event,
	l pb.UI_Settings_Level,
	w *strings.Builder,
) {
	// The event header can be written in different ways depending on the
	// what information we have about the scenario and the operation. It could
	// look like any of the following:
	//
	// ScenarioName Operation: OpStatus
	// ScenarioName [variant:value variant2:value] Operation: OpStatus

	scenario := flightplan.NewScenario()
	scenario.FromRef(event.GetOp().GetScenario())

	w.WriteString(fmt.Sprintf("%s %s: %s",
		scenario.String(),
		action,
		v.opStatusString(event.GetStatus()),
	))
}
