package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// newScenarioGenerateCmd returns a new 'scenario generate' sub-command
func newScenarioGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "generate [FILTER]",
		Short:             "Generate Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Generate a Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioGenerateCmd,
		Args:              scenarioFilterArgs,
		ValidArgsFunction: scenarioNameCompletion,
		Hidden:            true, // This is hidden because it is intended for debug only
	}

	return cmd
}

// runScenarioGenerateCmd is the function that generates scenarios
func runScenarioGenerateCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	res, err := enosClient.GenerateScenarios(ctx, &pb.GenerateScenariosRequest{
		Workspace: &pb.Workspace{
			Flightplan: flightPlan,
			OutDir:     scenarioCfg.outDir,
		},
		Filter: sf.Proto(),
	})
	if err != nil {
		return err
	}

	return ui.ShowScenarioGenerate(res)
}
