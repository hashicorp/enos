package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func newScenarioValidateConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "validate [FILTER]",
		Short:             "Validate configuration",
		Long:              "Validate all scenario and variant configurations",
		RunE:              runScenarioValidateCfgCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}
}

// runScenarioValidateCfgCmd is the function that validates all flight plan configuration
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
			Filter: sf.Proto(),
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowScenariosValidateConfig(res)
}
