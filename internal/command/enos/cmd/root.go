package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui"
	"github.com/hashicorp/hcl/v2"
)

var rootCmd = &cobra.Command{
	Use:               "enos",
	Short:             "Enos is your friendly neighborhood test runner",
	Long:              "Enos is a one stop shop for defining and executing complex test scenarios",
	PersistentPreRun:  rootCmdPreRun,
	SilenceErrors:     true, // we handle this ourselves
	CompletionOptions: cobra.CompletionOptions{DisableDescriptions: true},
}

var rootArgs struct {
	levelS string
}

// UI is our default CLI UI
var UI *ui.UI

// Execute executes enos
func Execute() {
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newScenarioCmd())
	rootCmd.PersistentFlags().StringVarP(&rootArgs.levelS, "log-level", "l", "info", "specify the log level")

	if err := rootCmd.Execute(); err != nil {
		diagErr, ok := err.(*errDiagnostic)
		if ok {
			err = UI.Diagnostics(diagErr.files, diagErr.diags)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		} else {
			UI.Error(err.Error())
		}

		os.Exit(1)
	}
}

func rootCmdPreRun(cmd *cobra.Command, args []string) {
	var level ui.Level
	switch rootArgs.levelS {
	case "warn":
		level = ui.WARN
	case "error":
		level = ui.ERROR
	default:
		level = ui.INFO
	}

	UI = ui.NewUI(
		ui.WithLevel(level),
		// NOTE: right now we only support text based output so we'll
		// always default to stdout, stderr, and stdin
		ui.WithStdin(os.Stdin),
		ui.WithStdout(os.Stdout),
		ui.WithStderr(os.Stderr),
	)
}

func decodeFlightPlan() (*flightplan.FlightPlan, error) {
	diags := hcl.Diagnostics{}

	var err error
	if baseDir != "" {
		baseDir, err = filepath.Abs(baseDir)
		if err != nil {
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unable to get absolute path from given working directory",
				Detail:   fmt.Sprintf("unable to get absolute path from given working directory: %s", err.Error()),
			})
		}
	}

	if baseDir == "" {
		baseDir, err = os.Getwd()
		if err != nil {
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unable to determine current working directory",
				Detail:   fmt.Sprintf("unable to determine current working directory: %s", err.Error()),
			})
		}
	}

	decoder, err := flightplan.NewDecoder(
		flightplan.WithDecoderBaseDir(baseDir),
	)
	if err != nil {
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to create new flight plan decoder",
			Detail:   fmt.Sprintf("unable to create new fligth plan decoder: %s", err.Error()),
		})
	}

	diags = diags.Extend(decoder.Parse())
	if diags.HasErrors() {
		return nil, diags
	}

	fp, moreDiags := decoder.Decode()
	diags = diags.Extend(moreDiags)

	if diags.HasErrors() {
		err := &errDiagnostic{diags: diags}
		if fp != nil {
			err.files = fp.Files
		}
		return fp, err
	}

	return fp, nil
}

type errDiagnostic struct {
	diags hcl.Diagnostics
	files map[string]*hcl.File
	err   error
}

// Error returns a joined message from all diagnostics errors
func (e *errDiagnostic) Error() string {
	msg := strings.Builder{}
	for i, err := range e.diags.Errs() {
		if i == 0 {
			msg.WriteString(err.Error())
		} else {
			msg.WriteString(fmt.Sprintf(": %s", err.Error()))
		}
	}

	return msg.String()
}

// Unwrap returns the wrapped error
func (e *errDiagnostic) Unwrap() error {
	return e.err
}
