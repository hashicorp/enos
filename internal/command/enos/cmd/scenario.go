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
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/operation/terraform"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/hcl/v2"
)

// scenarioConfig is the 'scenario' sub-command configuration type
type scenarioStateType struct {
	baseDir        string
	outDir         string
	fp             *flightplan.FlightPlan
	protoFp        *pb.FlightPlan
	timeout        time.Duration
	tfConfig       *terraform.Config
	lockTimeout    time.Duration
	varsFilesPaths []string
}

// scenarioState is the 'scenario' sub-command configuration
var scenarioState = scenarioStateType{
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
		Use:               "scenario",
		Short:             "Enos quality requirement scenarios",
		Long:              "Enos quality requirement scenarios",
		PersistentPreRunE: scenarioCmdPreRun,
		PersistentPostRun: scenarioCmdPostRun,
	}

	scenarioCmd.PersistentFlags().DurationVar(&scenarioState.timeout, "timeout", 15*time.Minute, "the command timeout")
	scenarioCmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.FailOnWarnings, "fail-on-warnings", false, "Fail immediately if warnings diagnostics are created")
	scenarioCmd.PersistentFlags().StringVarP(&scenarioState.baseDir, "chdir", "d", "", "use the given directory as the working directory")
	scenarioCmd.PersistentFlags().StringVarP(&scenarioState.outDir, "out", "o", "", "base directory where generated modules will be created")
	scenarioCmd.PersistentFlags().StringSliceVar(&scenarioState.varsFilesPaths, "var-file", []string{}, "path to use for variable values files (default will load all enos*.vars.hcl files in the working directory)")

	scenarioCmd.AddCommand(newScenarioListCmd())
	scenarioCmd.AddCommand(newScenarioGenerateCmd())
	scenarioCmd.AddCommand(newScenarioCheckCmd())
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
	err := rootCmdPreRun(cmd, args)
	if err != nil {
		return err
	}

	// Convert arguments that cobra flags can't handle
	scenarioState.tfConfig.Flags.LockTimeout = durationpb.New(scenarioState.lockTimeout)

	// Determine our default configuration
	err = setupDefaultScenarioCfg()
	if err != nil {
		return err
	}

	// Load the configuration from our working dir
	scenarioState.protoFp, err = readFlightPlanConfig(scenarioState.baseDir, scenarioState.varsFilesPaths)
	return err
}

// scenarioCmdPostRun is the scenario sub-command post-run. We'll use it to shut
// down the server.
func scenarioCmdPostRun(cmd *cobra.Command, args []string) {
	if rootState.enosServer != nil {
		err := rootState.enosServer.Stop()
		if err != nil {
			_ = ui.ShowError(err)
		}
	}
}

// setupDefaultScenarioCfg sets up default scenario configuration
func setupDefaultScenarioCfg() error {
	var err error

	if scenarioState.baseDir != "" {
		scenarioState.baseDir, err = filepath.Abs(scenarioState.baseDir)
		if err != nil {
			return fmt.Errorf("unable to get absolute path from given working directory: %w", err)
		}
	} else {
		scenarioState.baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("unable to determine current working directory: %w", err)
		}
	}

	if scenarioState.outDir != "" {
		scenarioState.outDir, err = filepath.Abs(scenarioState.outDir)
		if err != nil {
			return fmt.Errorf("unable to get absolute path from given working directory: %w", err)
		}
	}

	return nil
}

// decodeFlightPlan decodes the flight plan
func decodeFlightPlan(cmd *cobra.Command) error {
	diags := hcl.Diagnostics{}

	pfp, err := readFlightPlanConfig(scenarioState.baseDir, scenarioState.varsFilesPaths)
	if err != nil {
		return err
	}

	decoder, err := flightplan.NewDecoder(
		flightplan.WithDecoderBaseDir(pfp.GetBaseDir()),
		flightplan.WithDecoderFPFiles(pfp.GetEnosHcl()),
		flightplan.WithDecoderVarFiles(pfp.GetEnosVarsHcl()),
		flightplan.WithDecoderEnv(pfp.GetEnosVarsEnv()),
	)
	if err != nil {
		return err
	}

	// At this point we don't need to pass usage because it's likely an issue
	// with the flight plan definition, not missing or invalid arguments.
	cmd.SilenceUsage = true

	diags = diags.Extend(decoder.Parse())
	if diags.HasErrors() {
		return &status.ErrDiagnostic{
			Diags: diagnostics.FromHCL(decoder.ParserFiles(), diags),
		}
	}

	fp, moreDiags := decoder.Decode()
	diags = diags.Extend(moreDiags)
	scenarioState.fp = fp

	if len(diags) > 0 {
		if !diags.HasErrors() && !scenarioState.tfConfig.FailOnWarnings {
			return nil
		}

		return &status.ErrDiagnostic{
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

	if len(scenarioState.fp.Scenarios) == 0 {
		return nil, cobra.ShellCompDirectiveDefault
	}

	names := []string{}
	for _, s := range scenarioState.fp.Scenarios {
		names = append(names, s.Name)
	}

	return names, cobra.ShellCompDirectiveDefault
}

// scenarioTimeoutContext returns a context and cancel function with the configured
// scenario timeout deadline.
func scenarioTimeoutContext() (context.Context, func()) {
	var cancel func()
	ctx := context.Background()
	if scenarioState.timeout != 0 {
		return context.WithTimeout(ctx, scenarioState.timeout)
	}

	return ctx, cancel
}

// readFlightPlanConfig scans a directory for Enos flight plan configuration and returns
// a new instance of FlightPlan.
func readFlightPlanConfig(dir string, varFilePaths []string) (*pb.FlightPlan, error) {
	fp := &pb.FlightPlan{
		BaseDir:     dir,
		EnosVarsEnv: os.Environ(),
	}

	cfgFiles, err := flightplan.FindRawFiles(dir, flightplan.FlightPlanFileNamePattern)
	if err != nil {
		return nil, err
	}

	var varsFiles flightplan.RawFiles
	if len(varFilePaths) == 0 {
		varsFiles, err = flightplan.FindRawFiles(dir, flightplan.VariablesNamePattern)
	} else {
		varsFiles, err = flightplan.LoadRawFiles(varFilePaths)
	}
	if err != nil {
		return nil, err
	}

	fp.EnosHcl = cfgFiles
	fp.EnosVarsHcl = varsFiles

	return fp, nil
}

// prepareScenarioOpReq takes commands args, parses them to build a filter, and
// returns a proto filter and proto workspace to use in requests.
func prepareScenarioOpReq(
	args []string,
) (
	*pb.Scenario_Filter,
	*pb.Workspace,
	error,
) {
	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		ui.ShowOperationEvent(&pb.Operation_Event{
			Diagnostics: diagnostics.FromErr(err),
			Value:       &pb.Operation_Event_Decode{},
		})
		return nil, nil, err
	}

	ws := &pb.Workspace{
		Flightplan: scenarioState.protoFp,
		OutDir:     scenarioState.outDir,
		TfExecCfg:  scenarioState.tfConfig.Proto(),
	}

	return sf.Proto(), ws, nil
}
