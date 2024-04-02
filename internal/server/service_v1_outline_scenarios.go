// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"cmp"
	"context"
	"slices"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// OutlineScenarios returns an outline of scenarios.
func (s *ServiceV1) OutlineScenarios(
	ctx context.Context,
	req *pb.OutlineScenariosRequest,
) (
	*pb.OutlineScenariosResponse,
	error,
) {
	res := &pb.OutlineScenariosResponse{}

	fp, decRes := flightplan.DecodeProto(
		ctx,
		req.GetWorkspace().GetFlightplan(),
		flightplan.DecodeTargetScenariosOutlines,
		req.GetFilter(),
	)
	res.Decode = decRes
	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		return res, nil
	}

	qualities := map[string]*pb.Quality{}
	for _, sb := range fp.ScenarioBlocks {
		if sb == nil || sb.Scenarios == nil || len(sb.Scenarios) < 1 {
			continue
		}
		scenario := sb.Scenarios[0]

		out := scenario.Outline()
		if out == nil {
			continue
		}
		out.Matrix = sb.DecodedMatrices.Original.Proto()

		res.Outlines = append(res.GetOutlines(), out)
		for _, qual := range out.GetVerifies() {
			qualities[qual.GetName()] = qual
		}
	}

	for _, qual := range qualities {
		res.Verifies = append(res.GetVerifies(), qual)
	}

	slices.SortStableFunc(res.GetVerifies(), func(a, b *pb.Quality) int {
		if n := cmp.Compare(a.GetName(), b.GetName()); n != 0 {
			return n
		}

		return cmp.Compare(a.GetDescription(), b.GetDescription())
	})

	return res, nil
}
