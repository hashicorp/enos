package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ListScenarios returns the version information.
func (s *ServiceV1) ListScenarios(
	ctx context.Context,
	req *pb.ListScenariosRequest,
) (
	*pb.ListScenariosResponse,
	error,
) {
	res := &pb.ListScenariosResponse{}

	fp, decRes := decodeFlightPlan(
		ctx,
		req.GetWorkspace().GetFlightplan(),
		flightplan.DecodeModeRef,
		req.GetFilter(),
	)
	res.Decode = decRes
	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		return res, nil
	}

	if len(fp.Scenarios) > 0 {
		res.Scenarios = []*pb.Ref_Scenario{}
		for _, s := range fp.Scenarios {
			res.Scenarios = append(res.Scenarios, s.Ref())
		}
	}

	return res, nil
}
