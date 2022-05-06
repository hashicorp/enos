package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/execute/terraform"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/server"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/hcl/v2"
)

// scenarioConfig is the 'scenario' sub-command configuration type
type scenarioConfig struct {
	baseDir     string
	outDir      string
	fp          *flightplan.FlightPlan
	timeout     time.Duration
	tfConfig    *terraform.Config
	lockTimeout time.Duration
}

var (
	flightPlan *pb.FlightPlan
	enosServer *server.ServiceV1
	enosClient pb.EnosServiceClient
)

// scenarioCfg is the 'scenario' sub-command configuration
var scenarioCfg = scenarioConfig{
	tfConfig: terraform.NewConfig(),
}

// scenarioFilterDesc scenario sub-command filter description
var scenarioFilterDesc = `

A SCENARIO FILTER or FILTER must be a single string value which
consists of a Scenario name and space separated and colon joined key value pairs
that specify VARIANT SUBFILTERS. Extra spaces between words are ignored. The first
word will be used as the Scenario name. If no name is present, the filter will
match all defined Scenarios. VARIANT SUBFILTERS will always consist of an optional
exclusive match with !, a variant key as a string, a colon, and value filter
where a * wildcard matches any character(s). As variants are not allowed to
include spaces, VARIANT SUBFILTERS cannot include spaces. E.g.

VARIANT SUBFILTER = '[!]KEY:PATTERN|WILDCARD|ABSOLUTE'

FILTER = '[SCENARIO NAME] [...VARIANT SUBFILTER]'`

// newScenarioCmd returns a new instance of the 'scenario' sub-command
func newScenarioCmd() *cobra.Command {
	scenarioCmd := &cobra.Command{
		Use:                "scenario",
		Short:              "Enos quality requirement scenarios",
		Long:               "Enos quality requirement scenarios",
		PersistentPreRunE:  scenarioCmdPreRun,
		PersistentPostRunE: scenarioCmdPostRun,
	}

	scenarioCmd.PersistentFlags().StringVarP(&scenarioCfg.baseDir, "chdir", "d", "", "use the given directory as the working directory")
	scenarioCmd.PersistentFlags().StringVarP(&scenarioCfg.outDir, "out", "o", "", "base directory where generated modules will be created")
	scenarioCmd.PersistentFlags().DurationVar(&scenarioCfg.timeout, "timeout", 15*time.Minute, "the command timeout")

	scenarioCmd.AddCommand(newScenarioListCmd())
	scenarioCmd.AddCommand(newScenarioGenerateCmd())
	scenarioCmd.AddCommand(newScenarioValidateCmd())
	scenarioCmd.AddCommand(newScenarioLaunchCmd())
	scenarioCmd.AddCommand(newScenarioDestroyCmd())
	scenarioCmd.AddCommand(newScenarioRunCmd())
	scenarioCmd.AddCommand(newScenarioExecCmd())
	scenarioCmd.AddCommand(newScenarioOutputCmd())

	return scenarioCmd
}

// scenarioCmdPreRun is the scenario sub-command pre-run. We'll use it to initialize
// the program and decode the enos flight plan.
func scenarioCmdPreRun(cmd *cobra.Command, args []string) error {
	var err error
	rootCmdPreRun(cmd, args)

	// Convert arguments that cobra flags can't handle
	scenarioCfg.tfConfig.Flags.LockTimeout = durationpb.New(scenarioCfg.lockTimeout)

	// Determine our default base directory and out directory
	err = setupDefaultScenarioCfg()
	if err != nil {
		return err
	}

	flightPlan, err = readFlightPlanConfig(scenarioCfg.baseDir)
	if err != nil {
		return err
	}

	enosServer, enosClient, err = startGRPCServer(context.Background(), 5*time.Second)
	if err != nil {
		return err
	}

	return decodeFlightPlan(cmd)
}

// scenarioCmdPostRun is the scenario sub-command post-run. We'll use it to shut
// down the server.
func scenarioCmdPostRun(cmd *cobra.Command, args []string) error {
	if enosServer != nil {
		enosServer.Stop()
	}

	return nil
}

// setupDefaultScenarioCfg sets up default scenario configuration
func setupDefaultScenarioCfg() error {
	var err error

	if scenarioCfg.baseDir != "" {
		scenarioCfg.baseDir, err = filepath.Abs(scenarioCfg.baseDir)
		if err != nil {
			return fmt.Errorf("unable to get absolute path from given working directory: %w", err)
		}
	} else {
		scenarioCfg.baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("unable to determine current working directory: %w", err)
		}
	}

	if scenarioCfg.outDir != "" {
		scenarioCfg.outDir, err = filepath.Abs(scenarioCfg.outDir)
		if err != nil {
			return fmt.Errorf("unable to get absolute path from given working directory: %w", err)
		}
	}

	return nil
}

// decodeFlightPlan decodes the flight plan
func decodeFlightPlan(cmd *cobra.Command) error {
	diags := hcl.Diagnostics{}

	decoder, err := flightplan.NewDecoder(
		flightplan.WithDecoderBaseDir(scenarioCfg.baseDir),
	)
	if err != nil {
		return fmt.Errorf("unable to create new flight plan decoder: %w", err)
	}

	// At this point we don't need to pass usage because it's likely an issue
	// with the flight plan definition, not missing or invalid arguments.
	cmd.SilenceUsage = true

	diags = diags.Extend(decoder.Parse())
	if diags.HasErrors() {
		return &flightplan.ErrDiagnostic{
			Diags: diagnostics.FromHCL(decoder.ParserFiles(), diags),
		}
	}

	fp, moreDiags := decoder.Decode()
	diags = diags.Extend(moreDiags)
	scenarioCfg.fp = fp

	if len(diags) > 0 {
		if rootArgs.noWarnings && !diags.HasErrors() {
			return nil
		}

		return &flightplan.ErrDiagnostic{
			Diags: diagnostics.FromHCL(decoder.ParserFiles(), diags),
		}
	}

	return nil
}

// scenarioNameCompletion returns a shell directive of available flight plans.
// For commands which operate on one or more scenarios we can use this to
// add double tab style completion.
func scenarioNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	err := decodeFlightPlan(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(scenarioCfg.fp.Scenarios) == 0 {
		return nil, cobra.ShellCompDirectiveDefault
	}

	names := []string{}
	for _, s := range scenarioCfg.fp.Scenarios {
		names = append(names, s.Name)
	}

	return names, cobra.ShellCompDirectiveDefault
}

// scenarioTimeoutContext returns a context and cancel function with the configured
// scenario timeout deadline.
func scenarioTimeoutContext() (context.Context, func()) {
	var cancel func()
	ctx := context.Background()
	if scenarioCfg.timeout != 0 {
		return context.WithTimeout(ctx, scenarioCfg.timeout)
	}

	return ctx, cancel
}

// scenarioFilterArgs is our own cobra.PositionsArgs implementation that
// validates that the arguments given to a scenario command are a valid scenario
// filter.
func scenarioFilterArgs(cmd *cobra.Command, args []string) error {
	_, err := flightplan.ParseScenarioFilter(args)
	return err
}

// readFlightPlanConfig scans a directory for Enos flight plan configuration and returns
// a new instance of FlightPlan.
func readFlightPlanConfig(dir string) (*pb.FlightPlan, error) {
	fp := &pb.FlightPlan{
		BaseDir: dir,
	}

	cfgFiles, err := flightplan.FindRawFiles(dir, flightplan.FlightPlanFileNamePattern)
	if err != nil {
		return nil, err
	}

	varsFiles, err := flightplan.FindRawFiles(dir, flightplan.VariablesNamePattern)
	if err != nil {
		return nil, err
	}

	fp.EnosHcl = cfgFiles
	fp.EnosVarsHcl = varsFiles

	return fp, nil
}
