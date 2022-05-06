package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
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
	cmd.PersistentFlags().Uint32Var(&scenarioCfg.tfConfig.Flags.Parallelism, "tf-parallelism", 10, "terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioCfg.lockTimeout, "lock-timeout", 1*time.Minute, "duration to wait for terraform lock")

	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioValidateCmd is the function that validates scenarios
func runScenarioValidateCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	res, err := enosClient.ValidateScenarios(ctx, &pb.ValidateScenariosRequest{
		Workspace: &pb.Workspace{
			Flightplan: flightPlan,
			OutDir:     scenarioCfg.outDir,
			TfExecCfg:  scenarioCfg.tfConfig.Proto(),
		},
		Filter: sf.Proto(),
	})
	if err != nil {
		return err
	}

	return ui.ShowScenarioValidate(res)
}
