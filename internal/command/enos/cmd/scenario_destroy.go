package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/internal/generate"
)

// newScenarioDestroyCmd returns a new `scenario destroy` sub-command
func newScenarioDestroyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "destroy [FILTER]",
		Short:             "Destroy previously generated Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Destroy previously generated Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioDestroyCmd,
		Args:              cobra.RangeArgs(0, 1),
		ValidArgsFunction: scenarioNameCompletion,
	}
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoLock, "no-lock", false, "don't wait for terraform state lock")
	cmd.PersistentFlags().IntVar(&scenarioCfg.tfConfig.Flags.Parallelism, "tf-parallelism", 10, "terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioCfg.tfConfig.Flags.LockTimeout, "lock-timeout", 1*time.Minute, "duration to wait for terraform lock")

	return cmd
}

// runScenarioDestroyCmd is the function that destroys scenarios
func runScenarioDestroyCmd(cmd *cobra.Command, args []string) error {
	return scenarioGenAndExec(args, func(ctx context.Context, gen *generate.Generator, exec *execute.Executor) error {
		return exec.Destroy(ctx)
	})
}
