package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/internal/generate"
)

// newScenarioRunCmd returns a new 'scenario run' command
func newScenarioRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run [FILTER]",
		Short:             "Run a Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Run a Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioRunCmd,
		Args:              cobra.RangeArgs(0, 1),
		ValidArgsFunction: scenarioNameCompletion,
	}
	cmd.PersistentFlags().BoolVarP(&scenarioCfg.tfConfig.Flags.AutoApprove, "auto-approve", "a", false, "auto-approve the destruction plan")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoColor, "no-color", false, "disable color output")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoLock, "no-lock", false, "don't wait for terraform state lock")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoInput, "no-input", false, "disable user input for missing fields")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoBackend, "no-backend", false, "disable the configured backend")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoDownload, "no-download", false, "disable downloading modules and providers")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoRefresh, "no-refresh", false, "disable refreshing state")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.RefreshOnly, "refresh-only", false, "only refresh state")
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.Upgrade, "upgrade", true, "upgrade modules and providers")
	// NOTE: Compact warnings should not be a factor when the UI UX has been implemented
	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.CompactWarnings, "compact-warnings", false, "show compact warnings")
	cmd.PersistentFlags().IntVar(&scenarioCfg.tfConfig.Flags.Parallelism, "scenario-parallelism", 10, "terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioCfg.tfConfig.Flags.LockTimeout, "lock-timeout", 1*time.Minute, "duration to wait for terraform lock")

	return cmd
}

// runScenarioRunCmd is the function that launchs scenarios
func runScenarioRunCmd(cmd *cobra.Command, args []string) error {
	return scenarioGenAndExec(args, func(ctx context.Context, gen *generate.Generator, exec *execute.Executor) error {
		err := gen.Generate()
		if err != nil {
			return err
		}
		return exec.Run(ctx)
	})
}
