// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"cmp"
	"context"
	"slices"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// ListSamples returns a list of samples.
func (s *ServiceV1) ListSamples(
	ctx context.Context,
	req *pb.ListSamplesRequest,
) (
	*pb.ListSamplesResponse,
	error,
) {
	res := &pb.ListSamplesResponse{}

	fp, decRes := flightplan.DecodeProto(
		ctx,
		req.GetWorkspace().GetFlightplan(),
		flightplan.DecodeTargetSamples,
		nil,
	)
	res.Decode = decRes
	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		return res, nil
	}

	if len(fp.Samples) > 0 {
		res.Samples = []*pb.Ref_Sample{}
		for _, s := range fp.Samples {
			res.Samples = append(res.GetSamples(), s.Ref())
		}

		slices.SortStableFunc(res.GetSamples(), func(a, b *pb.Ref_Sample) int {
			return cmp.Compare(a.GetId().GetName(), b.GetId().GetName())
		})
	}

	return res, nil
}
