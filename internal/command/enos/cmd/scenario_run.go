package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// newScenarioRunCmd returns a new 'scenario run' sub-command
func newScenarioRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run [FILTER]",
		Short:             "Run Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Run Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioRunCmd,
		Args:              scenarioFilterArgs,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().BoolVar(&scenarioCfg.tfConfig.Flags.NoLock, "no-lock", false, "Don't wait for the Terraform state lock")
	cmd.PersistentFlags().Uint32Var(&scenarioCfg.tfConfig.Flags.Parallelism, "tf-parallelism", 10, "The Terraform scenario parallelism")
	cmd.PersistentFlags().DurationVar(&scenarioCfg.lockTimeout, "lock-timeout", 1*time.Minute, "The Duration to wait for the Terraform lock")

	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioRunCmd is the function that runs scenarios
func runScenarioRunCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	res, err := enosClient.RunScenarios(ctx, &pb.RunScenariosRequest{
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

	return ui.ShowScenarioRun(res)
}
