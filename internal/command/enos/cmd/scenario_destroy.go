package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// newScenarioDestroyCmd returns a new `scenario destroy` sub-command
func newScenarioDestroyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "destroy [FILTER]",
		Short:             "Destroy previously generated Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Destroy previously generated Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioDestroyCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoLock, "no-lock", false, "Don't wait for Terraform state lock")
	cmd.PersistentFlags().Uint32Var(&scenarioState.tfConfig.Flags.Parallelism, "tf-parallelism", 10, "The Terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioState.lockTimeout, "lock-timeout", 1*time.Minute, "The duration to wait for a Terraform lock")

	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioDestroyCmd is the function that destroys scenarios
func runScenarioDestroyCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, ws, err := prepareScenarioOpReq(args)
	if err != nil {
		return err
	}

	res, err := rootState.enosConnection.Client.DestroyScenarios(
		ctx, &pb.DestroyScenariosRequest{
			Workspace: ws,
			Filter:    sf,
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowOperationResponses(rootState.enosConnection.StreamOperations(ctx, res, ui))
}
