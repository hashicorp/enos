package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/internal/generate"
)

// newScenarioValidateCmd returns a new 'scenario validate' sub-command
func newScenarioValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "validate [FILTER]",
		Short:             "Validates a Terraform module from matching scenarios",
		Long:              fmt.Sprintf("Validates a Terraform module from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioValidateCmd,
		Args:              scenarioFilterArgs,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoLock, "no-lock", false, "don't wait for terraform state lock")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoBackend, "no-backend", false, "disable the configured backend")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoDownload, "no-download", false, "disable downloading modules and providers")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoRefresh, "no-refresh", false, "disable refreshing state")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.RefreshOnly, "refresh-only", false, "only refresh state")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.Upgrade, "upgrade", false, "upgrade modules and providers")
	cmd.PersistentFlags().IntVar(&scenarioCfg.tfConfig.Flags.Parallelism, "tf-parallelism", 10, "terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioCfg.tfConfig.Flags.LockTimeout, "lock-timeout", 1*time.Minute, "duration to wait for terraform lock")

	return cmd
}

// runScenarioValidateCmd is the function that validates scenarios
func runScenarioValidateCmd(cmd *cobra.Command, args []string) error {
	return scenarioGenAndExec(args, func(ctx context.Context, gen *generate.Generator, exec *execute.Executor) error {
		err := gen.Generate()
		if err != nil {
			return err
		}

		_, err = exec.Validate(ctx)
		return err
	})
}
