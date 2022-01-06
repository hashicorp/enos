package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/internal/generate"
)

func newScenarioExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "exec [FILTER] --cmd TERRAFORM-SUB-COMMAND",
		Short:             "Exec a Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Exec a Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioExecCmd,
		Args:              cobra.RangeArgs(0, 1),
		ValidArgsFunction: scenarioNameCompletion,
	}
	cmd.PersistentFlags().StringVar(&scenarioCfg.tfConfig.ExecSubCmd, "cmd", "", "the terraform sub-command")
	_ = cmd.MarkFlagRequired("cmd")

	return cmd
}

// runScenarioExecCmd is the function that launchs scenarios
func runScenarioExecCmd(cmd *cobra.Command, args []string) error {
	return scenarioGenAndExec(args, func(ctx context.Context, gen *generate.Generator, exec *execute.Executor) error {
		_, err := exec.Exec(ctx)
		return err
	})
}
