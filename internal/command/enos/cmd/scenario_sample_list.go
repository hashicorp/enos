// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// NewScenarioSampleListCmd returns a new scenario samples list command.
func NewScenarioSampleListCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "list",
		Short:             "List samples",
		Long:              "List all samples",
		RunE:              runSampleListCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}
}

// runSampleListCmd runs a scenario list.
func runSampleListCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	res, err := rootState.enosConnection.Client.ListSamples(
		ctx, &pb.ListSamplesRequest{
			Workspace: &pb.Workspace{
				Flightplan: scenarioState.protoFp,
			},
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowSampleList(res)
}
