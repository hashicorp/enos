package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/diagnostics"
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
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoLock, "no-lock", false, "Don't wait for the Terraform state lock")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoBackend, "no-backend", false, "Disable the configured backend")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoDownload, "no-download", false, "Disable downloading modules and providers")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoRefresh, "no-refresh", false, "Disable refreshing state")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.RefreshOnly, "refresh-only", false, "Only refresh state")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.Upgrade, "upgrade", false, "Upgrade modules and providers")
	cmd.PersistentFlags().Uint32Var(&scenarioState.tfConfig.Flags.Parallelism, "tf-parallelism", 10, "Terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioState.lockTimeout, "lock-timeout", 1*time.Minute, "Duration to wait for the Terraform lock")

	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioValidateCmd is the function that validates scenarios
func runScenarioValidateCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return ui.ShowScenarioValidate(&pb.ValidateScenariosResponse{
			Decode: &pb.Scenario_Operation_Decode_Response{
				Diagnostics: diagnostics.FromErr(err),
			},
		})
	}

	res, err := rootState.enosClient.ValidateScenarios(ctx, &pb.ValidateScenariosRequest{
		Workspace: &pb.Workspace{
			Flightplan: scenarioState.protoFp,
			OutDir:     scenarioState.outDir,
			TfExecCfg:  scenarioState.tfConfig.Proto(),
		},
		Filter: sf.Proto(),
	})
	if err != nil {
		return err
	}

	return ui.ShowScenarioValidate(res)
}
