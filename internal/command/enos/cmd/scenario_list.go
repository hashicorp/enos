// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// newScenarioListCmd returns a new 'scenario list' sub-command.
func newScenarioListCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "list [FILTER]",
		Short:             "List scenarios",
		Long:              "List all scenario and variant combinations",
		RunE:              runScenarioListCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}
}

// runScenarioListCmd runs a scenario list.
func runScenarioListCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return ui.ShowScenarioList(&pb.ListScenariosResponse{
			Diagnostics: diagnostics.FromErr(err),
		})
	}

	stream, err := rootState.enosConnection.Client.ListScenarios(
		ctx, &pb.ListScenariosRequest{
			Workspace: &pb.Workspace{
				Flightplan: scenarioState.protoFp,
			},
			Filter: sf.Proto(),
		},
	)
	if err != nil {
		return err
	}

	res := &pb.ListScenariosResponse{}
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch val := msg.GetResponse().(type) {
		case *pb.EnosServiceListScenariosResponse_Decode:
			res.Decode = val.Decode
		case *pb.EnosServiceListScenariosResponse_Scenario:
			res.Scenarios = append(res.Scenarios, val.Scenario)
		default:
		}
	}

	return ui.ShowScenarioList(res)
}
