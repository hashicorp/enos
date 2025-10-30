// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

type sampleObserveFilter struct {
	SampleName     string
	OnlySubsets    []string
	ExcludeSubsets []string
	Max            int32
	Min            int32
	Pct            float32
	Seed           int64
}

func (t *sampleObserveFilter) Proto() *pb.Sample_Filter {
	f := &pb.Sample_Filter{
		Sample:      &pb.Ref_Sample{Id: &pb.Sample_ID{Name: t.SampleName}},
		MaxElements: t.Max,
		MinElements: t.Min,
		Percentage:  t.Pct,
		Seed:        t.Seed,
	}

	for i := range t.OnlySubsets {
		if i == 0 {
			f.Subsets = []*pb.Sample_Subset_ID{}
		}
		f.Subsets = append(f.GetSubsets(), &pb.Sample_Subset_ID{Name: t.OnlySubsets[i]})
	}

	for i := range t.ExcludeSubsets {
		if i == 0 {
			f.ExcludeSubsets = []*pb.Sample_Subset_ID{}
		}
		f.ExcludeSubsets = append(f.GetExcludeSubsets(), &pb.Sample_Subset_ID{Name: t.ExcludeSubsets[i]})
	}

	return f
}

// NewScenarioSampleObserveCmd returns a new 'scenario samples observe' sub-command.
func NewScenarioSampleObserveCmd() *cobra.Command {
	sampleObserveCmd := &cobra.Command{
		Use:   "observe [sample_name] [args]",
		Short: "Take an observation of the scenario sample",
		Long:  `Take an observation of the scenario sample. This returns a list of all the possible scenarios/variant included in the subsets for the sample (also known as the sample frame). The observation must be limited to a particular sample by passing the sample name as an argument. The sample frame can be limited by using --include or --exclude flags. The number of randomly selected scenarios to observe can be limited using the min (minimum number of scenarios elements to return), max (maximum number of scenarios elements to return), and pct (limit then the overall possible scenarios as a percentage of total scenarios included in the frame) flags. If the max is set to a negative number, it will set no default upper bound. If a pct is set, it will create an upper bound of the percentage of the total sample frame. If both a max and a pct are set, whichever is lower will be used as the upper bound. By default, the min is set to 1 unless otherwise specified. If a min is set that is higher than the actual number of scenarios in the frame, an error will be returned. If replicable sample is desired (you can execute the observe command and get the same results) an entropy seed can be used to control the random number source. If no seed is given a random one will be chosen for you.`,
		RunE:  runSampleShowCmd,
		Args:  cobra.ExactArgs(1), // The sample name
	}

	sampleObserveCmd.PersistentFlags().StringSliceVarP(&scenarioState.sampleFilter.OnlySubsets, "include", "i", nil, "Limit the sample frame to the given subset(s)")
	sampleObserveCmd.PersistentFlags().StringSliceVarP(&scenarioState.sampleFilter.ExcludeSubsets, "exclude", "e", nil, "Exclude the given subset(s) from the sample frame")
	sampleObserveCmd.PersistentFlags().Int32Var(&scenarioState.sampleFilter.Min, "min", 1, "The minimum number of sample elements to return")
	sampleObserveCmd.PersistentFlags().Int32Var(&scenarioState.sampleFilter.Max, "max", -1, "The maximum number of sample elements to return")
	sampleObserveCmd.PersistentFlags().Float32Var(&scenarioState.sampleFilter.Pct, "pct", -1, "The percentage of sample elements to return")
	sampleObserveCmd.PersistentFlags().Int64Var(&scenarioState.sampleFilter.Seed, "seed", -1, "The entropy seed for the sampling random source")

	return sampleObserveCmd
}

// runSampleShowCmd runs a scenario list.
func runSampleShowCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := scenarioTimeoutContext()
	defer cancel()

	scenarioState.sampleFilter.SampleName = args[0]

	res, err := rootState.enosConnection.Client.ObserveSample(
		ctx, &pb.ObserveSampleRequest{
			Workspace: &pb.Workspace{
				Flightplan: scenarioState.protoFp,
			},
			Filter: scenarioState.sampleFilter.Proto(),
		},
	)
	if err != nil {
		return err
	}

	return ui.ShowSampleObservation(res)
}
