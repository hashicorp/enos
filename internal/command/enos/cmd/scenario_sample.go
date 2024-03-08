// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newScenarioCmd returns a new instance of the 'scenario' sub-command.
func newScenarioSampleCmd() *cobra.Command {
	sampleCmd := &cobra.Command{
		Use:   "sample",
		Short: "Enos scenario samples",
		Long:  "Enos scenario samples",
		Args: func(cmd *cobra.Command, args []string) error {
			if scenarioState.sampleFilter.Pct > 0 && scenarioState.sampleFilter.Pct > 100 {
				return fmt.Errorf("sampling percentage must be between 1 and 100, got %f", scenarioState.sampleFilter.Pct)
			}

			return nil
		},
	}

	sampleCmd.AddCommand(NewScenarioSampleListCmd())
	sampleCmd.AddCommand(NewScenarioSampleObserveCmd())

	return sampleCmd
}
