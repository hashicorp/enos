package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// newScenarioRunCmd returns a new 'scenario run' sub-command
func newScenarioRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run [FILTER]",
		Short:             "Run Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Run Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioRunCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoLock, "no-lock", false, "Don't wait for the Terraform state lock")
	cmd.PersistentFlags().Uint32Var(&scenarioState.tfConfig.Flags.Parallelism, "tf-parallelism", 10, "The Terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioState.lockTimeout, "lock-timeout", 1*time.Minute, "The Duration to wait for the Terraform lock")

	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioRunCmd is the function that runs scenarios
func runScenarioRunCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, ws, err := prepareScenarioOpReq(args)
	if err != nil {
		return err
	}

	res, err := rootState.enosConnection.Client.RunScenarios(
		ctx, &pb.RunScenariosRequest{
			Workspace: ws,
			Filter:    sf,
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowOperationResponses(rootState.enosConnection.StreamOperations(ctx, res, ws, ui))
}
