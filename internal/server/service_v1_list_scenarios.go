// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ListScenarios returns a list of scenarios and their variants.
func (s *ServiceV1) ListScenarios(
	ctx context.Context,
	req *pb.ListScenariosRequest,
) (
	*pb.ListScenariosResponse,
	error,
) {
	res := &pb.ListScenariosResponse{}

	fp, decRes := flightplan.DecodeProto(
		ctx,
		req.GetWorkspace().GetFlightplan(),
		flightplan.DecodeTargetScenariosNamesExpandVariants,
		req.GetFilter(),
	)
	res.Decode = decRes
	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		return res, nil
	}

	scenarios := fp.Scenarios()
	if len(scenarios) > 0 {
		res.Scenarios = []*pb.Ref_Scenario{}
		for _, s := range scenarios {
			res.Scenarios = append(res.GetScenarios(), s.Ref())
		}
	}

	return res, nil
}
