package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/internal/generate"
)

// newScenarioGenerateCmd returns a new 'scenario generate' sub-command
func newScenarioGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "generate [FILTER]",
		Short:             "Generate a Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Generate a Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioGenerateCmd,
		Args:              cobra.RangeArgs(0, 1),
		ValidArgsFunction: scenarioNameCompletion,
	}

	return cmd
}

// runScenarioGenerateCmd is the function that generates scenarios
func runScenarioGenerateCmd(cmd *cobra.Command, args []string) error {
	return scenarioGenAndExec(args, func(ctx context.Context, gen *generate.Generator, exec *execute.Executor) error {
		return gen.Generate()
	})
}
