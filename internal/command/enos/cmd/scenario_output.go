package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/internal/execute/terraform/format"
	"github.com/hashicorp/enos/internal/generate"
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

	return cmd
}

// runScenarioOutputCmd is the function that returns scenario output
func runScenarioOutputCmd(cmd *cobra.Command, args []string) error {
	return scenarioGenAndExec(args, func(ctx context.Context, gen *generate.Generator, exec *execute.Executor) error {
		res, err := exec.Output(ctx)
		if err != nil {
			return err
		}

		if scenarioCfg.tfConfig.OutputName != "" {
			out, ok := res[scenarioCfg.tfConfig.OutputName]
			if !ok {
				return fmt.Errorf("no output value for %s found", scenarioCfg.tfConfig.OutputName)
			}

			s, err := format.TerraformOutput(out, 2)
			if err != nil {
				return err
			}
			UI.Output(gen.Scenario.String())
			UI.Output(fmt.Sprintf("  %s\n", s))
			return nil
		}

		UI.Output(gen.Scenario.String())
		for name, out := range res {
			s, err := format.TerraformOutput(out, 2)
			if err != nil {
				return err
			}
			UI.Output(fmt.Sprintf("  %s = %s", name, s))
		}

		return nil
	})
}
