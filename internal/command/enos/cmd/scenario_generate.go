// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// newScenarioGenerateCmd returns a new 'scenario generate' sub-command.
func newScenarioGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "generate [FILTER]",
		Short:             "Generate Terraform modules from matching scenarios",
		Long:              "Generate a Terraform modules from matching scenarios. " + scenarioFilterDesc,
		RunE:              runScenarioGenerateCmd,
		ValidArgsFunction: scenarioNameCompletion,
		Hidden:            true, // This is hidden because it is intended for debug only
	}

	return cmd
}

// runScenarioGenerateCmd is the function that generates scenarios.
func runScenarioGenerateCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, ws, err := prepareScenarioOpReq(args)
	if err != nil {
		return err
	}

	res, err := rootState.enosConnection.Client.GenerateScenarios(
		ctx, &pb.GenerateScenariosRequest{
			Workspace: ws,
			Filter:    sf,
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowOperationResponses(rootState.enosConnection.StreamOperations(ctx, res, ui))
}
