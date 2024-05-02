// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func newScenarioValidateConfigCmd() *cobra.Command {
	scenarioValidateCmd := &cobra.Command{
		Use:               "validate [SCENARIO FILTER] <args>",
		Short:             "Validate configuration",
		Long:              "Validate enos flightplan configuration",
		RunE:              runScenarioValidateCfgCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}

	scenarioValidateCmd.PersistentFlags().StringSliceVarP(&scenarioState.sampleFilter.OnlySubsets, "include", "i", nil, "Limit the sample frame to the given subset(s)")
	scenarioValidateCmd.PersistentFlags().StringSliceVarP(&scenarioState.sampleFilter.ExcludeSubsets, "exclude", "e", nil, "Exclude the given subset(s) from the sample frame")
	scenarioValidateCmd.PersistentFlags().StringVarP(&scenarioState.sampleFilter.SampleName, "sample-name", "s", "", "Focus on a sample by name")
	scenarioValidateCmd.PersistentFlags().BoolVar(&scenarioState.noValidateScenarios, "no-scenarios", false, "Do not validate scenarios")
	scenarioValidateCmd.PersistentFlags().BoolVar(&scenarioState.noValidateSamples, "no-samples", false, "Do not validate scenario samples")

	return scenarioValidateCmd
}

// runScenarioValidateCfgCmd is the function that validates all flight plan configuration.
func runScenarioValidateCfgCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, err := flightplan.ParseScenarioFilter(args)
	if err != nil {
		return ui.ShowScenariosValidateConfig(&pb.ValidateScenariosConfigurationResponse{
			Diagnostics: diagnostics.FromErr(err),
		})
	}

	res, err := rootState.enosConnection.Client.ValidateScenariosConfiguration(
		ctx, &pb.ValidateScenariosConfigurationRequest{
			Workspace: &pb.Workspace{
				Flightplan: scenarioState.protoFp,
			},
			Filter:              sf.Proto(),
			SampleFilter:        scenarioState.sampleFilter.Proto(),
			NoValidateSamples:   scenarioState.noValidateSamples,
			NoValidateScenarios: scenarioState.noValidateScenarios,
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowScenariosValidateConfig(res)
}
