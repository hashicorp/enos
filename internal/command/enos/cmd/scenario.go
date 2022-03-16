package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/internal/execute/terraform"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/generate"
	"github.com/hashicorp/hcl/v2"
)

// scenarioConfig is the 'scenario' sub-command configuration type
type scenarioConfig struct {
	baseDir  string
	outDir   string
	fp       *flightplan.FlightPlan
	timeout  time.Duration
	tfConfig *terraform.Config
}

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

FILTER = '[SCENARIO NAME] [...VARIANT SUBFILTER]'

NOTE: VARIANT SUBFILTERS have not been implemented yet`

// newScenarioCmd returns a new instance of the 'scenario' sub-command
func newScenarioCmd() *cobra.Command {
	scenarioCmd := &cobra.Command{
		Use:               "scenario",
		Short:             "Enos quality requirement scenarios",
		Long:              "Enos quality requirement scenarios",
		PersistentPreRunE: scenarioCmdPreRun,
	}

	scenarioCmd.PersistentFlags().StringVarP(&scenarioCfg.baseDir, "chdir", "d", "", "use the given directory as the working directory")
	scenarioCmd.PersistentFlags().StringVarP(&scenarioCfg.outDir, "out", "o", "", "base directory for scenario state")
	scenarioCmd.PersistentFlags().DurationVar(&scenarioCfg.timeout, "timeout", 15*time.Minute, "the command timeout")

	scenarioCmd.AddCommand(newScenarioListCmd())
	scenarioCmd.AddCommand(newScenarioGenerateCmd())
	scenarioCmd.AddCommand(newScenarioValidateCmd())
	scenarioCmd.AddCommand(newScenarioLaunchCmd())
	scenarioCmd.AddCommand(newScenarioDestroyCmd())
	scenarioCmd.AddCommand(newScenarioRunCmd())
	scenarioCmd.AddCommand(newScenarioExecCmd())

	return scenarioCmd
}

// scenarioCmdPreRun is the scenario sub-command pre-run. We'll use it to initialize
// the program and decode the enos flight plan.
func scenarioCmdPreRun(cmd *cobra.Command, args []string) error {
	rootCmdPreRun(cmd, args)

	return decodeFlightPlan(cmd)
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
	} else {
		scenarioCfg.outDir = filepath.Join(scenarioCfg.baseDir, ".enos")
	}

	return nil
}

// decodeFlightPlan decodes the flight plan
func decodeFlightPlan(cmd *cobra.Command) error {
	diags := hcl.Diagnostics{}

	err := setupDefaultScenarioCfg()
	if err != nil {
		return err
	}

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
			Files: decoder.ParserFiles(),
			Diags: diags,
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
			Files: decoder.ParserFiles(),
			Diags: diags,
		}
	}

	return nil
}

// filterScenarios takes CLI arguments that may contain a scenario filter and
// returns the filtered results.
func filterScenarios(args []string) ([]*flightplan.Scenario, error) {
	filterArg := ""
	if len(args) == 1 {
		filterArg = args[0]
	}

	filter, err := flightplan.NewScenarioFilter(
		flightplan.WithScenarioFilterParse(filterArg),
	)
	if err != nil {
		return nil, err
	}

	return scenarioCfg.fp.ScenariosSelect(filter), nil
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

// newGeneratorFor returns a generator for a given scenario, its base directory
// and the generated module out directory.
func newGeneratorFor(scenario *flightplan.Scenario, baseDir string, outDir string) (*generate.Generator, error) {
	var err error
	var gen *generate.Generator

	outDir, err = filepath.Abs(outDir)
	if err != nil {
		return gen, err
	}

	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		return gen, err
	}

	return generate.NewGenerator(
		generate.WithScenario(scenario),
		generate.WithScenarioBaseDirectory(baseDir),
		generate.WithOutBaseDirectory(outDir),
		generate.WithUI(UI),
	)
}

// newExecutorFor takes an existing generator and returns an executor configured
// to execute what the generator will generate.
func newExecutorFor(gen *generate.Generator) (*execute.Executor, error) {
	// get a copy of our terraform CLI configuration and populate it with
	// the generators output paths and files.
	tfCfg := *scenarioCfg.tfConfig
	tfCfg.ConfigPath = gen.TerraformRCPath()
	tfCfg.DirPath = gen.TerraformModuleDir()
	tfCfg.UI = UI

	if gen != nil && gen.Scenario != nil {
		if gen.Scenario.TerraformCLI != nil {
			tfCfg.Env = gen.Scenario.TerraformCLI.Env

			if gen.Scenario.TerraformCLI.Path != "" {
				path, err := filepath.Abs(gen.Scenario.TerraformCLI.Path)
				if err != nil {
					return nil, fmt.Errorf("expanding path to terraform binary: %w", err)
				}
				tfCfg.BinPath = path
			}
		}
	}

	return execute.NewExecutor(
		execute.WithTerraformConfig(&tfCfg),
	)
}

// scenarioGenAndExec takes the command arguments, filters the scenarios that
// match the given filter and executes the given function with the generator
// and executor.
func scenarioGenAndExec(args []string, f func(context.Context, *generate.Generator, *execute.Executor) error) error {
	scenarios, err := filterScenarios(args)
	if err != nil {
		return err
	}

	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	for _, scenario := range scenarios {
		gen, err := newGeneratorFor(scenario, scenarioCfg.baseDir, scenarioCfg.outDir)
		if err != nil {
			return err
		}

		exec, err := newExecutorFor(gen)
		if err != nil {
			return err
		}

		err = f(ctx, gen, exec)
		if err != nil {
			return err
		}
	}

	return nil
}
