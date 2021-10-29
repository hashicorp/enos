package cmd

import (
	"github.com/spf13/cobra"
)

func newScenarioCmd() *cobra.Command {
	scenarioCmd := &cobra.Command{
		Use:   "scenario",
		Short: "Enos quality requirement scenarios",
		Long:  "Enos quality requirement scenarios",
	}

	scenarioCmd.AddCommand(newScenarioListCmd())

	return scenarioCmd
}
