package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/hashicorp/enos/internal/client"
	"github.com/hashicorp/enos/internal/server"
	uipkg "github.com/hashicorp/enos/internal/ui"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

var rootCmd = &cobra.Command{
	Use:               "enos",
	Short:             "Enos is a tool for powering Software Quality as Code",
	Long:              "Enos is a tool for powering Software Quality as Code by writing Terraform-based quality requirement scenarios using a composable, modular, and declarative language",
	PersistentPreRunE: rootCmdPreRun,
	PersistentPostRun: rootCmdPostRun,
	CompletionOptions: cobra.CompletionOptions{DisableDescriptions: true},
}

type rootStateS struct {
	logLevel       string // client log level
	logLevelServer string // server log level
	listenGRPC     string
	format         string
	stderrPath     string
	stdoutPath     string
	enosServer     *server.ServiceV1
	enosConnection *client.Connection
	operatorConfig *pb.Operator_Config
	profile        bool
	cpuProfileOut  io.ReadWriteCloser
}

var rootState = &rootStateS{
	operatorConfig: &pb.Operator_Config{},
}

// ui is our default CLI UI for things that have not been migrated to use
// the view.
var ui uipkg.View

// Execute executes enos.
func Execute() {
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newScenarioCmd())
	rootCmd.AddCommand(newFmtCmd())

	rootCmd.PersistentFlags().StringVar(&rootState.logLevel, "log-level", "info", "The log level for client output. Supported levels are error, warn, info, debug, and trace.")
	rootCmd.PersistentFlags().StringVar(&rootState.logLevelServer, "server-log-level", "error", "The log level for server output. Supported leves are error, warn, info, and debug")
	rootCmd.PersistentFlags().StringVar(&rootState.listenGRPC, "listen-grpc", "http://localhost:3205", "The gRPC server listen address")
	rootCmd.PersistentFlags().StringVar(&rootState.format, "format", "text", "The output format to use: text or json")
	rootCmd.PersistentFlags().StringVar(&rootState.stdoutPath, "stdout", "", "The path to write output. (default $STDOUT)")
	rootCmd.PersistentFlags().StringVar(&rootState.stderrPath, "stderr", "", "The path to write error output. (default $STDERR)")
	rootCmd.PersistentFlags().Int32Var(&rootState.operatorConfig.WorkerCount, "worker-count", 4, "The number of scenario operation workers")
	rootCmd.PersistentFlags().BoolVar(&rootState.profile, "profile", false, "Enable Go profiling")
	_ = rootCmd.PersistentFlags().MarkHidden("profile")

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
		FailOnWarnings: scenarioState.tfConfig.FailOnWarnings,
	}

	if rootState.format == "json" {
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

	if rootState.stderrPath != "" {
		uiCfg.StderrPath = rootState.stderrPath
	}

	if rootState.stdoutPath != "" {
		uiCfg.StdoutPath = rootState.stdoutPath
	}

	switch strings.ToLower(rootState.logLevel) {
	case "trace", "t", "a":
		uiCfg.Level = pb.UI_Settings_LEVEL_TRACE
	case "debug", "d":
		uiCfg.Level = pb.UI_Settings_LEVEL_DEBUG
	case "error", "e":
		uiCfg.Level = pb.UI_Settings_LEVEL_ERROR
	case "warn", "w":
		uiCfg.Level = pb.UI_Settings_LEVEL_WARN
	default:
		uiCfg.Level = pb.UI_Settings_LEVEL_INFO
	}

	var err error
	ui, err = uipkg.New(uiCfg)

	return err
}

func startCPUProfiling() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	rootState.cpuProfileOut, err = os.Create(filepath.Join(wd, "cpu.pprof"))
	if err != nil {
		return err
	}

	if err := pprof.StartCPUProfile(rootState.cpuProfileOut); err != nil {
		return err
	}

	return nil
}

func runMemoryProfiling() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	m, err := os.Create(filepath.Join(wd, "memory.pprof"))
	if err != nil {
		return err
	}
	defer m.Close()

	runtime.GC()

	if err := pprof.WriteHeapProfile(m); err != nil {
		return err
	}

	return nil
}

func rootCmdPreRun(cmd *cobra.Command, args []string) error {
	cmd.SilenceErrors = true // we handle this ourselves

	if rootState.profile {
		if err := startCPUProfiling(); err != nil {
			return err
		}
	}

	// Setup our UI configuration first
	err := setupCLIUI()
	if err != nil {
		return err
	}

	// If we're this far they've given use valid usage and we'll handle it
	cmd.SilenceUsage = true

	// Create our gRPC server and client
	rootState.enosServer, rootState.enosConnection, err = startServer(
		context.Background(),
		5*time.Second,
	)
	if err != nil {
		return err
	}

	return err
}

func rootCmdPostRun(cmd *cobra.Command, args []string) {
	if rootState.profile {
		if rootState.cpuProfileOut != nil {
			defer rootState.cpuProfileOut.Close()
		}
		defer pprof.StopCPUProfile()
	}

	if rootState.enosServer != nil {
		err := rootState.enosServer.Stop()
		if err != nil {
			_ = ui.ShowError(err)
		}
	}

	// Run memory profiling after we've shut everything down everything but
	// our UI
	if rootState.profile {
		if err := runMemoryProfiling(); err != nil {
			_ = ui.ShowError(err)
		}
	}

	ui.Close()
}
