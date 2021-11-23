package cmd

import (
	"github.com/spf13/cobra"
)

var baseDir string

func newScenarioCmd() *cobra.Command {
	scenarioCmd := &cobra.Command{
		Use:   "scenario",
		Short: "Enos quality requirement scenarios",
		Long:  "Enos quality requirement scenarios",
	}

	scenarioCmd.PersistentFlags().StringVarP(&baseDir, "chdir", "d", "", "use the given directory as the working directory")

	scenarioCmd.AddCommand(newScenarioListCmd())
	scenarioCmd.AddCommand(newScenarioGenerateCmd())

	return scenarioCmd
}
