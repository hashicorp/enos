package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func newScenarioExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "exec [FILTER] [ARGS] --cmd TERRAFORM-SUB-COMMAND",
		Short:             "Execute a terraform sub-command in the context of previously generated Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Execute a terraform sub-command in the context of previously generated Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioExecCmd,
		Args:              scenarioFilterArgs,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().StringVar(&scenarioCfg.tfConfig.ExecSubCmd, "cmd", "", "The Terraform sub-command")

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
		return status.Error(codes.InvalidArgument, err.Error())
	}

	res, err := enosClient.ExecScenarios(ctx, &pb.ExecScenariosRequest{
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

	return ui.ShowScenarioExec(res)
}
