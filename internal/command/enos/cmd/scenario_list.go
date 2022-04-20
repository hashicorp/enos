package cmd

import (
	"github.com/spf13/cobra"
)

// newScenarioListCmd returns a new 'scenario list' sub-command
func newScenarioListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [FILTER]",
		Short: "List scenarios",
		Long:  "List scenarios",
		RunE:  runScenarioListCmd,
	}
}

// runScenarioListCmd runs a scenario list
func runScenarioListCmd(cmd *cobra.Command, args []string) error {
	scenarios, err := filterScenarios(args)
	if err != nil {
		return err
	}

	if len(scenarios) == 0 {
		return nil
	}

	header := []string{"scenario"}
	rows := [][]string{{""}} // add a padding row
	for _, scenario := range scenarios {
		rows = append(rows, []string{scenario.String()})
	}

	UI.RenderTable(header, rows)

	return nil
}
