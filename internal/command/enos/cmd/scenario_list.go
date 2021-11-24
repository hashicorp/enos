package cmd

import (
	"sort"

	"github.com/spf13/cobra"
)

func newScenarioListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Enos quality requirement scenarios",
		Long:  "List Enos quality requirement scenarios",
		RunE:  runScenarioListCmd,
	}
}

func runScenarioListCmd(cmd *cobra.Command, args []string) error {
	fp, err := decodeFlightPlan()
	if err != nil {
		cmd.SilenceUsage = true
		return err
	}

	if len(fp.Scenarios) != 0 {
		sort.Slice(fp.Scenarios, func(i, j int) bool {
			return fp.Scenarios[i].Name < fp.Scenarios[j].Name
		})
		header := []string{"scenario"}
		rows := [][]string{{""}} // add a padding row
		for _, scenario := range fp.Scenarios {
			rows = append(rows, []string{scenario.Name})
		}

		UI.RenderTable(header, rows)
	}

	return nil
}
