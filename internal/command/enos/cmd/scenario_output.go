package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/diagnostics"
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
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().StringVar(&scenarioState.tfConfig.OutputName, "name", "", "The Terraform state value to show")

	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioOutputCmd is the function that returns scenario output
func runScenarioOutputCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return ui.ShowScenarioOutput(&pb.OutputScenariosResponse{
			Decode: &pb.Scenario_Operation_Decode_Response{
				Diagnostics: diagnostics.FromErr(err),
			},
		})
	}

	res, err := rootState.enosClient.OutputScenarios(ctx, &pb.OutputScenariosRequest{
		Workspace: &pb.Workspace{
			Flightplan: scenarioState.protoFp,
			OutDir:     scenarioState.outDir,
			TfExecCfg:  scenarioState.tfConfig.Proto(),
		},
		Filter: sf.Proto(),
	})
	if err != nil {
		return err
	}

	return ui.ShowScenarioOutput(res)
}
