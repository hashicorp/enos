// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// newScenarioOutlineCmd returns a new 'scenario outline' sub-command.
func newScenarioOutlineCmd() *cobra.Command {
	outlineCmd := &cobra.Command{
		Use:               "outline [NAME]",
		Short:             "Provide an outline of the function of scenarios",
		Long:              "Provide an outline of the function of scenarios. This includes the description, matrix variants, steps. and quality verifications that are performed",
		RunE:              runScenarioOutlineCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}

	outlineCmd.PersistentFlags().StringVar(&rootState.format, "format", "text", "The output format to use: text, json, html")

	return outlineCmd
}

// runScenarioOutlineCmd runs a scenario outline.
func runScenarioOutlineCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return ui.ShowScenarioOutline(&pb.OutlineScenariosResponse{
			Diagnostics: diagnostics.FromErr(err),
		})
	}

	res, err := rootState.enosConnection.Client.OutlineScenarios(
		ctx, &pb.OutlineScenariosRequest{
			Workspace: &pb.Workspace{
				Flightplan: scenarioState.protoFp,
			},
			Filter: sf.Proto(),
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowScenarioOutline(res)
}
