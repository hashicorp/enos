package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// newScenarioListCmd returns a new 'scenario list' sub-command
func newScenarioListCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "list [FILTER]",
		Short:             "List scenarios",
		Long:              "List all scenario and variant combinations",
		RunE:              runScenarioListCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}
}

// runScenarioListCmd runs a scenario list
func runScenarioListCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return ui.ShowScenarioList(&pb.ListScenariosResponse{
			Diagnostics: diagnostics.FromErr(err),
		})
	}

	res, err := rootState.enosConnection.Client.ListScenarios(
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

	return ui.ShowScenarioList(res)
}
