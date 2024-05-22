// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

func newScenarioExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "exec [FILTER] [ARGS] --cmd TERRAFORM-SUB-COMMAND",
		Short:             "Execute a terraform sub-command in the context of previously generated Terraform modules from matching scenarios",
		Long:              "Execute a terraform sub-command in the context of previously generated Terraform modules from matching scenarios. " + scenarioFilterDesc,
		RunE:              runScenarioExecCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().StringVar(&scenarioState.tfConfig.ExecSubCmd, "cmd", "", "The Terraform sub-command")

	_ = cmd.MarkFlagRequired("cmd")
	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioExecCmd is the function that launchs scenarios.
func runScenarioExecCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, ws, err := prepareScenarioOpReq(args)
	if err != nil {
		return err
	}

	res, err := rootState.enosConnection.Client.ExecScenarios(
		ctx, &pb.ExecScenariosRequest{
			Workspace: ws,
			Filter:    sf,
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowOperationResponses(rootState.enosConnection.StreamOperations(ctx, res, ui))
}
