// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"time"

	"github.com/spf13/cobra"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// newScenarioCheckCmd returns a new 'scenario check' sub-command.
func newScenarioCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "check [FILTER]",
		Short:             "Check that scenarios are valid",
		Long:              "Check that scenarios are valid by generating the Scenario's Terraform Root Module, initializing it, validating it, and planning. " + scenarioFilterDesc,
		RunE:              runScenarioCheckCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoLock, "no-lock", false, "Don't wait for the Terraform state lock")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoBackend, "no-backend", false, "Disable the configured backend")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoDownload, "no-download", false, "Disable downloading modules and providers")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoRefresh, "no-refresh", false, "Disable refreshing state")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.RefreshOnly, "refresh-only", false, "Only refresh state")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.Upgrade, "upgrade", false, "Upgrade modules and providers")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoReconfigure, "no-reconfigure", false, "Don't reconfigure the backend during init")
	cmd.PersistentFlags().Uint32Var(&scenarioState.tfConfig.Flags.Parallelism, "tf-parallelism", 10, "Terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioState.lockTimeout, "lock-timeout", 1*time.Minute, "Duration to wait for the Terraform lock")

	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioCheckCmd is the function that checks scenarios.
func runScenarioCheckCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, ws, err := prepareScenarioOpReq(args)
	if err != nil {
		return err
	}

	res, err := rootState.enosConnection.Client.CheckScenarios(
		ctx,
		&pb.CheckScenariosRequest{
			Workspace: ws,
			Filter:    sf,
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowOperationResponses(rootState.enosConnection.StreamOperations(ctx, res, ui))
}
