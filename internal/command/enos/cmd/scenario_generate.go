package cmd

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/flightplan/generate"
)

var (
	scenarioGenerateOutDir string
	scenarioFilterDesc     = `A SCENARIO FILTER or FILTER must be a single string value which
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
)

func newScenarioGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "generate [FILTER]",
		Short:             "Generate a Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Generate a Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioGenerateCmd,
		Args:              cobra.RangeArgs(0, 1),
		ValidArgsFunction: scenarioGenerateCompletion,
	}

	cmd.PersistentFlags().StringVarP(&scenarioGenerateOutDir, "out", "o", "enos.out", "base directory to use for generated modules")

	return cmd
}

// scenarioGenerateCompletion adds double tab style completion support to generate
// by creating a list of scenario filters that are available.
func scenarioGenerateCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	fp, err := decodeFlightPlan()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(fp.Scenarios) == 0 {
		return nil, cobra.ShellCompDirectiveDefault
	}

	names := []string{}
	sort.Slice(fp.Scenarios, func(i, j int) bool {
		return fp.Scenarios[i].Name < fp.Scenarios[j].Name
	})
	for _, s := range fp.Scenarios {
		names = append(names, s.Name)
	}

	return names, cobra.ShellCompDirectiveDefault
}

// runScenarioGenerateCmd is the function that generates scenarios
func runScenarioGenerateCmd(cmd *cobra.Command, args []string) error {
	fp, err := decodeFlightPlan()
	if err != nil {
		cmd.SilenceUsage = true
		return err
	}

	filterArg := ""
	if len(args) == 1 {
		filterArg = args[0]
	}

	filter, err := flightplan.NewScenarioFilter(
		flightplan.WithScenarioFilterParse(filterArg),
	)
	if err != nil {
		return err
	}

	outDir, err := filepath.Abs(scenarioGenerateOutDir)
	if err != nil {
		return err
	}

	for _, scenario := range fp.ScenariosSelect(filter) {
		gen, err := generate.NewGenerator(
			generate.WithScenario(scenario),
			generate.WithBaseDirectory(baseDir),
			generate.WithOutDirectory(outDir),
			generate.WithUI(UI),
		)
		if err != nil {
			return err
		}

		err = gen.Generate()
		if err != nil {
			return err
		}
	}

	return nil
}
