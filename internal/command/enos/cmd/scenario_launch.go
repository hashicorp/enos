package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/internal/generate"
)

// newScenarioLaunchCmd returns a new 'scenario launch' sub-command
func newScenarioLaunchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "launch [FILTER]",
		Short:             "Launch a Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Launch a Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioLaunchCmd,
		Args:              cobra.RangeArgs(0, 1),
		ValidArgsFunction: scenarioNameCompletion,
	}
	cmd.PersistentFlags().BoolVarP(&scenarioCfg.tfConfig.Flags.AutoApprove, "auto-approve", "a", false, "auto-approve the destruction plan")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoLock, "no-lock", false, "don't wait for terraform state lock")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoInput, "no-input", false, "disable user input for missing fields")
	// NOTE: Compact warnings should not be a factor when the UI UX has been implemented
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.CompactWarnings, "compact-warnings", false, "show compact warnings")
	cmd.PersistentFlags().IntVar(&scenarioCfg.tfConfig.Flags.Parallelism, "scenario-parallelism", 10, "terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioCfg.tfConfig.Flags.LockTimeout, "lock-timeout", 1*time.Minute, "duration to wait for terraform lock")

	return cmd
}

// runScenarioLaunchCmd is the function that launchs scenarios
func runScenarioLaunchCmd(cmd *cobra.Command, args []string) error {
	return scenarioGenAndExec(args, func(ctx context.Context, gen *generate.Generator, exec *execute.Executor) error {
		return exec.Launch(ctx)
	})
}
