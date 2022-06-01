package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/hashicorp/enos/internal/flightplan"
	uipkg "github.com/hashicorp/enos/internal/ui"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

var rootCmd = &cobra.Command{
	Use:               "enos",
	Short:             "Enos is your friendly neighborhood test runner",
	Long:              "Enos is a one stop shop for defining and executing complex test scenarios",
	PersistentPreRun:  rootCmdPreRun,
	PersistentPostRun: rootCmdPostRun,
	SilenceErrors:     true, // we handle this ourselves
	CompletionOptions: cobra.CompletionOptions{DisableDescriptions: true},
}

var rootArgs struct {
	logLevelC  string // client log level
	logLevelS  string // server log level
	noWarnings bool
	listenGRPC string
	format     string
	stderrPath string
	stdoutPath string
}

// ui is our default CLI UI for things that have not been migrated to use
// the view.
var ui uipkg.View

// Execute executes enos
func Execute() {
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newScenarioCmd())
	rootCmd.PersistentFlags().StringVar(&rootArgs.logLevelC, "client-log-level", "info", "specify the log level for client output")
	rootCmd.PersistentFlags().StringVar(&rootArgs.logLevelS, "server-log-level", "error", "specify the log level for server output")
	rootCmd.PersistentFlags().BoolVar(&rootArgs.noWarnings, "silence-warnings", false, "silence warnings")
	rootCmd.PersistentFlags().StringVar(&rootArgs.listenGRPC, "listen-grpc", "http://localhost:3205", "the gRPC server listen address")
	rootCmd.PersistentFlags().StringVar(&rootArgs.format, "format", "text", "the output format to use: text or json")
	rootCmd.PersistentFlags().StringVar(&rootArgs.stdoutPath, "out", "", "the path to write output. If unset it uses STDOUT")
	rootCmd.PersistentFlags().StringVar(&rootArgs.stderrPath, "error-out", "", "the path to write error output. If unset it uses STDERR")

	if err := rootCmd.Execute(); err != nil {
		diagErr, ok := err.(*flightplan.ErrDiagnostic)
		if ok {
			err = ui.ShowDiagnostics(diagErr.Diags)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		} else {
			err2 := ui.ShowError(err)
			if err2 != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", err.Error(), err2.Error())
			}
		}

		os.Exit(1)
	}
}

func setupCLIUI() error {
	uiCfg := &pb.UI_Settings{
		Width:  78,
		Format: pb.UI_Settings_FORMAT_BASIC_TEXT,
	}

	if rootArgs.format == "json" {
		uiCfg.Format = pb.UI_Settings_FORMAT_JSON
	}

	if term.IsTerminal(int(os.Stdout.Fd())) {
		uiCfg.IsTty = true
		uiCfg.UseColor = true
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			uiCfg.Width = uint32(width)
		}
	}

	if rootArgs.stderrPath != "" {
		uiCfg.StderrPath = rootArgs.stderrPath
	}

	if rootArgs.stdoutPath != "" {
		uiCfg.StdoutPath = rootArgs.stdoutPath
	}

	switch rootArgs.logLevelC {
	case "debug", "DEBUG", "Debug", "d":
		uiCfg.Level = pb.UI_Settings_LEVEL_DEBUG
	case "error", "ERROR", "Error", "e", "err":
		uiCfg.Level = pb.UI_Settings_LEVEL_ERROR
	case "warn", "WARN", "Warn", "w":
		uiCfg.Level = pb.UI_Settings_LEVEL_WARN
	default:
		uiCfg.Level = pb.UI_Settings_LEVEL_INFO
	}

	var err error
	ui, err = uipkg.New(uiCfg)

	return err
}

func rootCmdPreRun(cmd *cobra.Command, args []string) {
	err := setupCLIUI()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func rootCmdPostRun(cmd *cobra.Command, args []string) {
	ui.Close()
}
