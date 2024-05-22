// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"
	"slices"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
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
		out.Matrix = sb.DecodedMatrices.GetOriginal().Proto()

		res.Outlines = append(res.GetOutlines(), out)
		for _, qual := range out.GetVerifies() {
			qualities[qual.GetName()] = qual
		}
	}

	verifies := []*pb.Quality{}
	for _, qual := range qualities {
		verifies = append(verifies, qual)
	}
	slices.SortStableFunc(verifies, flightplan.CompareQualityProto)

	res.Verifies = verifies

	return res, nil
}
