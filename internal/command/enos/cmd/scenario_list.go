package cmd

import (
	"github.com/spf13/cobra"
)

// newScenarioListCmd returns a new 'scenario list' sub-command
func newScenarioListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Enos quality requirement scenarios",
		Long:  "List Enos quality requirement scenarios",
		RunE:  runScenarioListCmd,
	}
}

// runScenarioListCmd runs a scenario list
func runScenarioListCmd(cmd *cobra.Command, args []string) error {
	if len(scenarioCfg.fp.Scenarios) != 0 {
		header := []string{"scenario"}
		rows := [][]string{{""}} // add a padding row
		for _, scenario := range scenarioCfg.fp.Scenarios {
			rows = append(rows, []string{scenario.Name})
		}

		UI.RenderTable(header, rows)
	}

	return nil
}
