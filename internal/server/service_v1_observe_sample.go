// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ObserveSample returns an observation of a sample.
func (s *ServiceV1) ObserveSample(
	ctx context.Context,
	req *pb.ObserveSampleRequest,
) (
	*pb.ObserveSampleResponse,
	error,
) {
	res := &pb.ObserveSampleResponse{}

	sampleReq, err := flightplan.NewSampleObservationReq(
		flightplan.WithSampleObservationReqWorkSpace(req.GetWorkspace()),
		flightplan.WithSampleObservationReqFilter(req.GetFilter()),
		flightplan.WithSampleObservationReqFunc(flightplan.SampleFuncPurposiveStratified),
	)
	if err != nil {
		res.Diagnostics = diagnostics.FromErr(err)

		return res, nil
	}

	res.Observation, res.Decode = sampleReq.Observe(ctx)

	return res, nil
}
