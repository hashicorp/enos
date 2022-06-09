package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func newScenarioExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "exec [FILTER] [ARGS] --cmd TERRAFORM-SUB-COMMAND",
		Short:             "Execute a terraform sub-command in the context of previously generated Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Execute a terraform sub-command in the context of previously generated Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioExecCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().StringVar(&scenarioState.tfConfig.ExecSubCmd, "cmd", "", "The Terraform sub-command")

	_ = cmd.MarkFlagRequired("cmd")
	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioExecCmd is the function that launchs scenarios
func runScenarioExecCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return ui.ShowScenarioExec(&pb.ExecScenariosResponse{
			Decode: &pb.Scenario_Operation_Decode_Response{
				Diagnostics: diagnostics.FromErr(err),
			},
		})
	}

	res, err := rootState.enosClient.ExecScenarios(ctx, &pb.ExecScenariosRequest{
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

	return ui.ShowScenarioExec(res)
}
