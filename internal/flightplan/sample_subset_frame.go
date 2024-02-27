// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/enos/internal/random"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// SampleSubsetFrame is filtered frame of a subset.
type SampleSubsetFrame struct {
	*SampleSubset
	ScenarioFilter *pb.Scenario_Filter
	*Matrix
}

// SampleSubsetFrames are a collection from one-or-more named subset frames.
type SampleSubsetFrames map[string]*SampleSubsetFrame // subset name -> subset frame

// Ref converts our frame to a wire reference. As it only refers to a frame it is lossy.
func (s *SampleSubsetFrame) Ref() *pb.Ref_Sample_Subset {
	if s.SampleSubset == nil {
		return nil
	}

	return &pb.Ref_Sample_Subset{
		Id: &pb.Sample_Subset_ID{
			Name: s.SampleSubset.Name,
		},
	}
}

// ObserveSimpleRandom takes a sample size and a randomness source and returns a sample subset
// observation from the sample subset using a simple random sampling strategy.
func (s *SampleSubsetFrame) ObserveSimpleRandom(take int32, r *rand.Rand) (*SampleSubsetObservation, error) {
	if take > s.Size() {
		return nil, fmt.Errorf("cannot take a sample of %d from subset frame of %d", take, s.Size())
	}

	if take == s.Size() {
		return &SampleSubsetObservation{
			SampleSubsetFrame: s,
			Matrix:            s.Matrix,
		}, nil
	}

	samples, err := random.SampleInt(int(take), len(s.Matrix.Vectors), r)
	if err != nil {
		return nil, err
	}

	nm := NewMatrix()
	for i := range samples {
		nm.AddVector(s.Matrix.Vectors[samples[i]])
	}

	return &SampleSubsetObservation{
		SampleSubsetFrame: s,
		Matrix:            nm,
	}, nil
}

// Size returns the total size of elements in the frame.
func (s *SampleSubsetFrame) Size() int32 {
	if s == nil {
		return 0
	}

	// Don't count blank frames.
	if s.SampleSubset == nil && s.Matrix == nil {
		return 0
	}

	// If we have a matrix count one element per vertex
	if s.Matrix != nil {
		return int32(len(s.Matrix.GetVectors()))
	}

	// We don't have a matrix so our size can only be a singular scenario
	return int32(1)
}

// Size returns the total size of elements in the frame.
func (s SampleSubsetFrames) Size() int32 {
	if s == nil || len(s) < 1 {
		return 0
	}

	size := int32(0)
	for _, v := range s {
		size += v.Size()
	}

	return size
}
