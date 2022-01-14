package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui"
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
	levelS     string
	noWarnings bool
}

// UI is our default CLI UI
var UI = ui.NewUI(
	ui.WithLevel(ui.ERROR),
	// NOTE: right now we only support text based output so we'll
	// always default to stdout, stderr, and stdin
	ui.WithStdin(os.Stdin),
	ui.WithStdout(os.Stdout),
	ui.WithStderr(os.Stderr),
)

// Execute executes enos
func Execute() {
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newScenarioCmd())
	rootCmd.PersistentFlags().StringVarP(&rootArgs.levelS, "log-level", "l", "info", "specify the log level")
	rootCmd.PersistentFlags().BoolVar(&rootArgs.noWarnings, "silence-warnings", false, "silence warnings")

	if err := rootCmd.Execute(); err != nil {
		diagErr, ok := err.(*flightplan.ErrDiagnostic)
		if ok {
			err = UI.Diagnostics(diagErr.Files, diagErr.Diags)
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
	switch rootArgs.levelS {
	case "warn":
		UI.Level = ui.WARN
	case "error":
		UI.Level = ui.ERROR
	default:
		UI.Level = ui.INFO
	}
}
