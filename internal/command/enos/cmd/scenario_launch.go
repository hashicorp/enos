// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// newScenarioLaunchCmd returns a new 'scenario run' command.
func newScenarioLaunchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "launch [FILTER]",
		Short:             "Apply previously validated Terraform modules from matching scenarios",
		Long:              "Apply previously validated Terraform modules from matching scenarios. " + scenarioFilterDesc,
		RunE:              runScenarioLaunchCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoLock, "no-lock", false, "Don't wait for the Terraform state lock")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoBackend, "no-backend", false, "Disable the configured backend")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoDownload, "no-download", false, "Disable downloading modules and providers")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoRefresh, "no-refresh", false, "Disable refreshing state")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.RefreshOnly, "refresh-only", false, "Only refresh state")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.Upgrade, "upgrade", true, "Upgrade modules and providers")
	cmd.PersistentFlags().BoolVar(&scenarioState.tfConfig.Flags.NoReconfigure, "no-reconfigure", false, "Don't reconfigure the backend during init")
	cmd.PersistentFlags().Uint32Var(&scenarioState.tfConfig.Flags.Parallelism, "tf-parallelism", 10, "The Terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioState.lockTimeout, "lock-timeout", 1*time.Minute, "The Duration to wait for the Terraform lock")

	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioLaunchCmd is the function that launches scenarios.
func runScenarioLaunchCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, ws, err := prepareScenarioOpReq(args)
	if err != nil {
		return err
	}

	res, err := rootState.enosConnection.Client.LaunchScenarios(
		ctx, &pb.LaunchScenariosRequest{
			Workspace: ws,
			Filter:    sf,
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowOperationResponses(rootState.enosConnection.StreamOperations(ctx, res, ui))
}
