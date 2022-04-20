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
		Use:               "exec [FILTER] [ARGS] --cmd TERRAFORM-SUB-COMMAND",
		Short:             "Execute a terraform sub-command in the context of previously generated Terraform modules from matching scenarios",
		Long:              fmt.Sprintf("Execute a terraform sub-command in the context of previously generated Terraform modules from matching scenarios. %s", scenarioFilterDesc),
		RunE:              runScenarioExecCmd,
		Args:              scenarioFilterArgs,
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
