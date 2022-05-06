package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// newScenarioListCmd returns a new 'scenario list' sub-command
func newScenarioListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [FILTER]",
		Short: "List scenarios",
		Long:  "List scenarios",
		RunE:  runScenarioListCmd,
	}
}

// runScenarioListCmd runs a scenario list
func runScenarioListCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	res, err := enosClient.ListScenarios(ctx, &pb.ListScenariosRequest{
		Workspace: &pb.Workspace{
			Flightplan: flightPlan,
		},
		Filter: sf.Proto(),
	})
	if err != nil {
		return err
	}

	return ui.ShowScenarioList(res)
}
