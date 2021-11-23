package ui

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/mitchellh/cli"
	"github.com/olekukonko/tablewriter"

	"github.com/hashicorp/hcl/v2"
)

// RenderTable does a basic render of table data to the desired writer
func (u *UI) RenderTable(header []string, rows [][]string) {
	table := tablewriter.NewWriter(u.Stdout)

	table.SetHeader(header)

	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetNoWhiteSpace(true)
	table.AppendBulk(rows)

	table.Render()
}

var _ cli.Ui = (*UI)(nil)

// UI is a CLI UI
type UI struct {
	Stderr io.Writer
	Stdout io.Writer
	Stdin  io.Reader

	AskPrefix       string
	AskSecretPrefix string
	OutputPrefix    string
	InfoPrefix      string
	ErrorPrefix     string
	WarnPrefix      string

	Level Level

	ui cli.Ui
}

// Level is the output level
type Level int

// Levels
const (
	INFO Level = iota
	WARN
	ERROR
)

// Opt is a UI option
type Opt func(*UI)

// NewUI takes zero or more options and returns a new UI
func NewUI(opts ...Opt) *UI {
	ui := &UI{
		Level:  INFO,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
	}
	for _, opt := range opts {
		opt(ui)
	}

	ui.ui = &cli.PrefixedUi{
		AskPrefix:       ui.AskPrefix,
		AskSecretPrefix: ui.AskSecretPrefix,
		OutputPrefix:    ui.OutputPrefix,
		InfoPrefix:      ui.InfoPrefix,
		ErrorPrefix:     ui.ErrorPrefix,
		WarnPrefix:      ui.WarnPrefix,
		Ui: &cli.BasicUi{
			Reader:      ui.Stdin,
			Writer:      ui.Stdout,
			ErrorWriter: ui.Stderr,
		},
	}

	return ui
}

// WithStderr sets stderr
func WithStderr(stderr io.Writer) Opt {
	return func(ui *UI) {
		ui.Stderr = stderr
	}
}

// WithStdout sets stdout
func WithStdout(stdout io.Writer) Opt {
	return func(ui *UI) {
		ui.Stdout = stdout
	}
}

// WithStdin sets stdin
func WithStdin(stdin io.Reader) Opt {
	return func(ui *UI) {
		ui.Stdin = stdin
	}
}

// WithAskPrefix sets the ask prefix
func WithAskPrefix(p string) Opt {
	return func(ui *UI) {
		ui.AskPrefix = p
	}
}

// WithAskSecretPrefix sets the ask prefix
func WithAskSecretPrefix(p string) Opt {
	return func(ui *UI) {
		ui.AskSecretPrefix = p
	}
}

// WithOutputPrefix sets the output prefix
func WithOutputPrefix(p string) Opt {
	return func(ui *UI) {
		ui.OutputPrefix = p
	}
}

// WithInfoPrefix sets the info prefix
func WithInfoPrefix(p string) Opt {
	return func(ui *UI) {
		ui.InfoPrefix = p
	}
}

// WithErrorPrefix sets the error prefix
func WithErrorPrefix(p string) Opt {
	return func(ui *UI) {
		ui.ErrorPrefix = p
	}
}

// WithWarnPrefix sets the warn prefix
func WithWarnPrefix(p string) Opt {
	return func(ui *UI) {
		ui.WarnPrefix = p
	}
}

// WithLevel sets logging level
func WithLevel(l Level) Opt {
	return func(ui *UI) {
		ui.Level = l
	}
}

// Ask prompts the user for some data
func (u *UI) Ask(q string) (string, error) {
	return u.ui.Ask(q)
}

// AskSecret prompts the user for some data
func (u *UI) AskSecret(q string) (string, error) {
	return u.ui.AskSecret(q)
}

// Output outputs a message to stdout
func (u *UI) Output(m string) {
	u.ui.Output(m)
}

// Info outputs a message at info level
func (u *UI) Info(m string) {
	if u.Level <= INFO {
		u.ui.Info(m)
	}
}

// Error outputs a message at error level
func (u *UI) Error(m string) {
	if u.Level <= ERROR {
		u.ui.Error(m)
	}
}

// Warn outputs a message at warn level
func (u *UI) Warn(m string) {
	if u.Level <= WARN {
		u.ui.Warn(m)
	}
}

// Diagnostics outputs diagnostics to stderr
func (u *UI) Diagnostics(files map[string]*hcl.File, diags hcl.Diagnostics) error {
	useColor := false
	if f, ok := u.Stderr.(*os.File); ok {
		if isatty.IsTerminal(f.Fd()) {
			useColor = true
		}
	}

	return hcl.NewDiagnosticTextWriter(
		u.Stderr,
		files,
		78, // wrap at
		useColor,
	).WriteDiagnostics(diags)
}
