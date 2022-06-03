package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	uipkg "github.com/hashicorp/enos/internal/ui"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

var rootCmd = &cobra.Command{
	Use:               "enos",
	Short:             "Enos is a tool for powering Software Quality as Code",
	Long:              "Enos is a tool for powering Software Quality as Code by writing Terraform-based quality requirement scenarios using a composable, modular, and declarative language",
	PersistentPreRun:  rootCmdPreRun,
	PersistentPostRun: rootCmdPostRun,
	CompletionOptions: cobra.CompletionOptions{DisableDescriptions: true},
}

var rootArgs struct {
	logLevel       string // client log level
	logLevelServer string // server log level
	listenGRPC     string
	format         string
	stderrPath     string
	stdoutPath     string
}

// ui is our default CLI UI for things that have not been migrated to use
// the view.
var ui uipkg.View

// Execute executes enos
func Execute() {
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newScenarioCmd())
	rootCmd.AddCommand(newFmtCmd())

	rootCmd.PersistentFlags().StringVar(&rootArgs.logLevel, "log-level", "info", "Log level for client output")
	rootCmd.PersistentFlags().StringVar(&rootArgs.logLevelServer, "server-log-level", "error", "The log level for server output")
	rootCmd.PersistentFlags().StringVar(&rootArgs.listenGRPC, "listen-grpc", "http://localhost:3205", "The gRPC server listen address")
	rootCmd.PersistentFlags().StringVar(&rootArgs.format, "format", "text", "Output format to use: text or json")
	rootCmd.PersistentFlags().StringVar(&rootArgs.stdoutPath, "out", "", "Path to write output. (default $STDOUT)")
	rootCmd.PersistentFlags().StringVar(&rootArgs.stderrPath, "error-out", "", "Path to write error output. (default $STDERR)")
	rootCmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.FailOnWarnings, "fail-on-warnings", false, "Fail immediately if warnings diagsnostics are created")

	if err := rootCmd.Execute(); err != nil {
		var exitErr *status.ErrExit
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode)
		}

		if ui != nil {
			var err2 error
			var diagErr *status.ErrDiagnostic
			if errors.As(err, &diagErr) {
				err2 = ui.ShowDiagnostics(diagErr.Diags)
			} else {
				err2 = ui.ShowError(err)
			}
			if err2 != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", err.Error(), err2.Error())
			}
		}

		os.Exit(1)
	}
}

func setupCLIUI() error {
	uiCfg := &pb.UI_Settings{
		Width:          78,
		Format:         pb.UI_Settings_FORMAT_BASIC_TEXT,
		FailOnWarnings: scenarioCfg.tfConfig.FailOnWarnings,
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

	switch rootArgs.logLevel {
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
	cmd.SilenceErrors = true // we handle this ourselves

	err := setupCLIUI()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	enosServer, enosClient, err = startGRPCServer(context.Background(), 5*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func rootCmdPostRun(cmd *cobra.Command, args []string) {
	ui.Close()
}
