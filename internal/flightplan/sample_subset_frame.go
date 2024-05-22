// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/hashicorp/enos/internal/random"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
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

	// If our sample subset has been configured with a matrix but we don't, that means that our
	// filter did not match any variants in subset frame. The most likely reason for this is that
	// we've configured our sample subset with invalid variants. Since there's not matching variants
	// our frame size is zero.
	if s.SampleSubset.Matrix != nil && len(s.SampleSubset.Matrix.Vectors) > 0 {
		return 0
	}

	// We don't have a matrix and neither does our subset so our size can only be a singular scenario
	// that doesn't include vaiants.
	return int32(1)
}

// Validate that a sample frame is capable of being used to sample from.
func (s *SampleSubsetFrame) Validate() error {
	if s == nil {
		return errors.New("sample subset frame has not been initialized")
	}

	if s.SampleSubset == nil {
		return errors.New("sample subset frame is missing reference to sample subset")
	}

	if s.Size() < 1 {
		msg := fmt.Sprintf("the sampling frame for %s/%s is invalid",
			s.SampleSubset.SampleName,
			s.SampleSubset.Name,
		)

		if s.SampleSubset != nil && s.SampleSubset.Matrix != nil && len(s.SampleSubset.Matrix.Vectors) > 0 {
			msg = fmt.Sprintf("%s: perhaps the matrix variants specified in the subset matrix exclude all possible combinations:\n%s",
				msg, s.SampleSubset.Matrix.String(),
			)
		}

		return errors.New(msg)
	}

	return nil
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
