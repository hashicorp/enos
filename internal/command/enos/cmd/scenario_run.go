package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/internal/generate"
)

// newScenarioRunCmd returns a new 'scenario run' sub-command
func newScenarioRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run [FILTER]",
		Short:             "Run Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Run Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioRunCmd,
		Args:              cobra.RangeArgs(0, 1),
		ValidArgsFunction: scenarioNameCompletion,
	}
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoLock, "no-lock", false, "don't wait for terraform state lock")
	cmd.PersistentFlags().IntVar(&scenarioCfg.tfConfig.Flags.Parallelism, "tf-parallelism", 10, "terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioCfg.tfConfig.Flags.LockTimeout, "lock-timeout", 1*time.Minute, "duration to wait for terraform lock")

	return cmd
}

// runScenarioRunCmd is the function that runs scenarios
func runScenarioRunCmd(cmd *cobra.Command, args []string) error {
	return scenarioGenAndExec(args, func(ctx context.Context, gen *generate.Generator, exec *execute.Executor) error {
		err := gen.Generate()
		if err != nil {
			return err
		}

		_, err = exec.Run(ctx)
		return err
	})
}
