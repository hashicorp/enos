package basic

import (
	"strings"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowDiagnostics writes a diagnostic to stderr.
func (v *View) ShowDiagnostics(diags []*pb.Diagnostic) error {
	v.WriteDiagnostics(diags)

	return nil
}

// WriteDiagnostics writes diagnostics in a basic human friendly way.
func (v *View) WriteDiagnostics(diags []*pb.Diagnostic) {
	if len(diags) < 1 {
		return
	}
	v.ui.Error(v.diagsToString(diags))
}

// diagsToString returns the diagsnostics as a string.
func (v *View) diagsToString(diags []*pb.Diagnostic) string {
	if len(diags) < 1 {
		return ""
	}

	b := new(strings.Builder)
	for _, diag := range diags {
		b.WriteString(diagnostics.String(
			diag,
			diagnostics.WithStringUISettings(v.settings),
			diagnostics.WithStringSnippetEnabled(true),
		))
	}

	return strings.TrimSpace(b.String())
}

func (v *View) writeDiags(d []*pb.Diagnostic, w *strings.Builder) {
	if len(d) < 1 {
		return
	}

	if diagnostics.HasErrors(d) || v.settings.GetLevel() >= pb.UI_Settings_LEVEL_WARN {
		w.WriteString("\n" + v.diagsToString(d))
	}
}
