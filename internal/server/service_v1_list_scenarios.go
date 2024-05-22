// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// ListScenarios returns a list of scenarios and their variants.
func (s *ServiceV1) ListScenarios(req *pb.ListScenariosRequest, stream pb.EnosService_ListScenariosServer) error {
	fp, decRes := flightplan.DecodeProto(
		stream.Context(),
		req.GetWorkspace().GetFlightplan(),
		flightplan.DecodeTargetScenariosNamesExpandVariants,
		req.GetFilter(),
	)

	err := stream.Send(&pb.EnosServiceListScenariosResponse{
		Response: &pb.EnosServiceListScenariosResponse_Decode{
			Decode: decRes,
		},
	})
	if err != nil {
		return err
	}

	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		// Short circuit if we've got a failure
		return nil
	}

	for _, scenario := range fp.Scenarios() {
		err := stream.Send(&pb.EnosServiceListScenariosResponse{
			Response: &pb.EnosServiceListScenariosResponse_Scenario{
				Scenario: scenario.Ref(),
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}
