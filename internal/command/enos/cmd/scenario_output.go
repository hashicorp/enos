package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// newScenarioOutputCmd returns a new 'scenario output' command
func newScenarioOutputCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "output [FILTER]",
		Short:             "Show the output of selected scenarios",
		Long:              fmt.Sprintf("Show the output of selected scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioOutputCmd,
		Args:              scenarioFilterArgs,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().StringVar(&scenarioCfg.tfConfig.OutputName, "name", "", "terraform state value to show")

	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioOutputCmd is the function that returns scenario output
func runScenarioOutputCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	res, err := enosClient.OutputScenarios(ctx, &pb.OutputScenariosRequest{
		Workspace: &pb.Workspace{
			Flightplan: flightPlan,
			OutDir:     scenarioCfg.outDir,
			TfExecCfg:  scenarioCfg.tfConfig.Proto(),
		},
		Filter: sf.Proto(),
	})
	if err != nil {
		return err
	}

	return ui.ShowScenarioOutput(res)
}
