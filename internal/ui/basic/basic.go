package basic

import (
	"io"
	"os"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/ui/terminal"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// View is our basic terminal CLI view
type View struct {
	settings *pb.UI_Settings
	ui       *terminal.UI
	stdout   io.ReadWriteCloser
	stderr   io.ReadWriteCloser
}

// Opt is a functional option
type Opt func(*View)

// New takes options and returns a new basic.View
func New(opts ...Opt) (*View, error) {
	v := &View{}

	for _, opt := range opts {
		opt(v)
	}

	uiOpts := []terminal.Opt{
		terminal.WithStdin(os.Stdin),
		terminal.WithStdout(os.Stdout),
		terminal.WithStderr(os.Stderr),
	}
	if v.settings != nil {
		uiOpts = append(uiOpts, terminal.WithLevel(v.settings.GetLevel()))

		if v.settings.GetIsTty() {
			uiOpts = append(uiOpts, terminal.WithColor(true))
		}

		if v.settings.GetWidth() > 0 {
			uiOpts = append(uiOpts, terminal.WithWidth(uint(v.settings.Width)))
		}

		if v.settings.GetStdoutPath() != "" {
			f, err := os.OpenFile(v.settings.GetStdoutPath(), os.O_RDWR|os.O_CREATE, 0o755)
			if err != nil {
				return nil, err
			}
			v.stdout = f

			uiOpts = append(uiOpts, terminal.WithStdout(f))
		}

		if v.settings.GetStderrPath() != "" {
			f, err := os.OpenFile(v.settings.GetStderrPath(), os.O_RDWR|os.O_CREATE, 0o755)
			if err != nil {
				return nil, err
			}
			v.stderr = f

			uiOpts = append(uiOpts, terminal.WithStderr(f))
		}

	}

	v.ui = terminal.NewUI(uiOpts...)

	return v, nil
}

// WithUISettings sets the ui settings
func WithUISettings(ui *pb.UI_Settings) Opt {
	return func(view *View) {
		view.settings = ui
	}
}

// Settings returns the views UI settings
func (v *View) Settings() *pb.UI_Settings {
	return v.settings
}

// ShowError writes the error message to stdout
func (v *View) ShowError(err error) error {
	v.ui.Error(err.Error())
	return nil
}

// ShowDiagnostics writes a diagnostic to stderr
func (v *View) ShowDiagnostics(diags []*pb.Diagnostic) error {
	v.WriteDiagnostics(diags)
	return nil
}

// Close closes any open file handles
func (v *View) Close() error {
	if v.stderr != nil {
		err := v.stderr.Close()
		if err != nil {
			return err
		}
	}

	if v.stdout == nil {
		return nil
	}

	return v.stdout.Close()
}

// WriteDiagnostics writes diagnostics in a basic human friendly way
func (v *View) WriteDiagnostics(diags []*pb.Diagnostic) {
	if len(diags) < 1 {
		return
	}

	for _, diag := range diags {
		v.ui.Error(diagnostics.String(
			diag,
			diagnostics.WithStringUISettings(v.settings),
			diagnostics.WithStringSnippetEnabled(true),
		))
	}
}
