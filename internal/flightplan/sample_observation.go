// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"time"

	"github.com/hashicorp/enos/internal/diagnostics"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// SampleObservationReq is a request to take an observation of a sample set.
type SampleObservationReq struct {
	Ws     *pb.Workspace
	Filter *pb.Sample_Filter
	Func   SampleObservationFunc
	Rand   *rand.Rand
}

// SampleObservationOpt is a functional option for a NewSampleObservationReq.
type SampleObservationOpt func(*SampleObservationReq)

// SampleObservation is a result of taking an algorithmic observation of a sample frame. Each subset
// is represented by a matrix whose vertices correspond to the sampled elements.
type SampleObservation struct {
	*SampleFrame       // The sample frame that was used to create the observation
	SubsetObservations SampleSubsetObservations
}

// SampleSubsetObservation is an obserservation of a sample subset frame.
type SampleSubsetObservation struct {
	*SampleSubsetFrame
	*Matrix
}

// SampleSubsetObservations are a map of subsets to their observations.
type SampleSubsetObservations map[string]*SampleSubsetObservation

// SampleObservationFunc takes a context, a sample frame, and a random number source and returns
// a SampleObservation.
type SampleObservationFunc func(context.Context, *SampleFrame, *rand.Rand) (*SampleObservation, error)

// NewSampleObservationReq takes optional NewSampleObservationOpt's and returns a new SampleObservationReq.
// Some validation is performed to ensure a valid request but not all validation can happen until
// the observation request is executed.
func NewSampleObservationReq(opts ...SampleObservationOpt) (*SampleObservationReq, error) {
	req := &SampleObservationReq{
		Func: SampleFuncPurposiveStratified,
	}

	for i := range opts {
		opts[i](req)
	}

	if req.Ws == nil {
		return nil, errors.New("cannot sample without a configured workspace")
	}

	if req.Ws.GetFlightplan() == nil {
		return nil, errors.New("cannot sample without a configured flightplan")
	}

	if req.Filter == nil {
		return nil, errors.New("cannot sample without a configured filter")
	}

	if req.Filter.GetSample().GetId().GetName() == "" {
		return nil, errors.New("cannot sample without a sample name in the filter")
	}

	// If we haven't configured a random number source then try and do it from the source seed.
	if req.Rand == nil {
		seed := req.Filter.GetSeed()
		if seed < 1 {
			seed = time.Now().UnixNano()
			req.Filter.Seed = seed
		}

		//nolint:gosec// G404 we're using a weak random number generator because secure random numbers
		// are not needed for this use case.
		req.Rand = rand.New(rand.NewSource(seed))
	}

	if req.Func == nil {
		return nil, errors.New("cannot sample without a configured sampling function")
	}

	return req, nil
}

func WithSampleObservationReqWorkSpace(ws *pb.Workspace) SampleObservationOpt {
	return func(req *SampleObservationReq) {
		req.Ws = ws
	}
}

func WithSampleObservationReqFilter(f *pb.Sample_Filter) SampleObservationOpt {
	return func(req *SampleObservationReq) {
		req.Filter = f
	}
}

func WithSampleObservationReqFunc(m SampleObservationFunc) SampleObservationOpt {
	return func(req *SampleObservationReq) {
		req.Func = m
	}
}

func WithSampleObservationReqRandSeed(seed int64) SampleObservationOpt {
	return func(req *SampleObservationReq) {
		//nolint:gosec // G404: we're using a weak random number generator to create
		// pseudorandom samples that can be deterministic if we're given a seed.
		req.Rand = rand.New(rand.NewSource(seed))
	}
}

// Observe returns a sample observation.
func (s *SampleObservationReq) Observe(ctx context.Context) (*pb.Sample_Observation, *pb.DecodeResponse) {
	res := &pb.Sample_Observation{
		Filter: s.Filter,
	}

	// Get the sample frame.
	frame, decRes := s.Frame(ctx)
	if diagnostics.HasFailed(
		s.Ws.GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		return res, decRes
	}
	if decRes == nil {
		decRes = &pb.DecodeResponse{}
	}

	// Get out sample observation.
	sampleObservation, err := s.Func(ctx, frame, s.Rand)
	if err != nil {
		decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromErr(err)...)
	}

	// Convert our observation to wire elements and expand attributes.
	res.Elements, err = sampleObservation.Elements(s.Rand)
	if err != nil {
		decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromErr(err)...)
	}

	return res, decRes
}

// Frame returns the valid sample Frame.
func (s *SampleObservationReq) Frame(ctx context.Context) (*SampleFrame, *pb.DecodeResponse) {
	return decodeAndGetSampleFrame(ctx, s.Ws, s.Filter)
}

// Size is the length of the our SubsetObservations.
func (s *SampleObservation) Size() int32 {
	if s == nil || len(s.SubsetObservations) < 1 {
		return 0
	}

	size := int32(0)
	for _, v := range s.SubsetObservations {
		size += v.Size()
	}

	return size
}

// Elements takes a random source, expands all of the sample elements in all frames, and returns
// the elements in the wire format.
func (s *SampleObservation) Elements(r *rand.Rand) ([]*pb.Sample_Element, error) {
	if s == nil || s.SampleFrame == nil || len(s.SubsetObservations) < 1 {
		return []*pb.Sample_Element{}, nil
	}

	res := []*pb.Sample_Element{}
	for _, name := range s.SubsetObservations.Keys() {
		subsetObsv, ok := s.SubsetObservations[name]
		if !ok {
			// This should never happen but because Keys() isn't a range over the collection directly we
			// could theoretically nil panic here if Keys() gives us invalid results.
			return nil, errors.New("failed to get observation elements for subset")
		}
		subElements, err := s.SampleFrame.Elements(subsetObsv.SampleSubset.Name, r, subsetObsv.Matrix)
		if err != nil {
			return nil, fmt.Errorf("attemping to get elements from subset %s: %w", subsetObsv.SampleSubset.Name, err)
		}

		res = append(res, subElements...)
	}

	// Make sure our elements are sorted by sample, subset, and scenario
	slices.SortStableFunc(res, func(a, b *pb.Sample_Element) int {
		if n := cmp.Compare(a.GetSample().GetId().GetName(), b.GetSample().GetId().GetName()); n != 0 {
			return n
		}

		if n := cmp.Compare(a.GetSubset().GetId().GetName(), b.GetSubset().GetId().GetName()); n != 0 {
			return n
		}

		return cmp.Compare(a.GetScenario().GetId().GetFilter(), b.GetScenario().GetId().GetFilter())
	})

	return res, nil
}

// Size is the size of the sample subset observation.
func (s *SampleSubsetObservation) Size() int32 {
	if s == nil {
		return 0
	}

	if s.Matrix == nil || len(s.Matrix.Vectors) < 1 {
		return 1
	}

	return int32(len(s.Matrix.Vectors))
}

// Size is the size of all sample subset observations.
func (s SampleSubsetObservations) Size() int32 {
	if len(s) < 1 {
		return 0
	}

	size := int32(0)
	for _, v := range s {
		size += v.Size()
	}

	return size
}

// Keys are sorted keys of the subset obserservation.
func (s SampleSubsetObservations) Keys() []string {
	if len(s) < 1 {
		return nil
	}

	keys := []string{}
	for k := range s {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	return keys
}
