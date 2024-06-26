// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	"github.com/hashicorp/go-multierror"
)

// newScenarioOutputCmd returns a new 'scenario output' command.
func newScenarioOutputCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "output [FILTER]",
		Short:             "Show the output of selected scenarios",
		Long:              "Show the output of selected scenarios. " + scenarioFilterDesc,
		RunE:              runScenarioOutputCmd,
		ValidArgsFunction: scenarioNameCompletion,
	}

	cmd.PersistentFlags().StringVar(&scenarioState.tfConfig.OutputName, "name", "", "The Terraform state value to show")

	_ = cmd.Flags().MarkHidden("out") // Allow passing out for testing but mark it hidden

	return cmd
}

// runScenarioOutputCmd is the function that returns scenario output.
func runScenarioOutputCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	sf, ws, err := prepareScenarioOpReq(args)
	if err != nil {
		return err
	}

	res, err := rootState.enosConnection.Client.OutputScenarios(
		ctx, &pb.OutputScenariosRequest{
			Workspace: ws,
			Filter:    sf,
		},
	)
	if err != nil {
		return err
	}

	// Stream the operations to wait until all outputs have been run and the
	// display them
	opRes := rootState.enosConnection.StreamOperations(ctx, res, ui)

	var merr *multierror.Error
	merr = multierror.Append(merr, ui.ShowDecode(opRes.GetDecode(), true))
	merr = multierror.Append(merr, ui.ShowOutput(opRes))

	return merr.ErrorOrNil()
}
