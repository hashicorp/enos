package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ListScenarios returns the version information
func (s *ServiceV1) ListScenarios(
	ctx context.Context,
	req *pb.ListScenariosRequest,
) (
	*pb.ListScenariosResponse,
	error,
) {
	res := &pb.ListScenariosResponse{}

	scenarios, diags := decodeAndFilter(req.GetWorkspace().GetFlightplan(), req.GetFilter())
	res.Diagnostics = diags
	if diagnostics.HasErrors(diags) {
		return res, nil
	}

	if len(scenarios) > 0 {
		res.Scenarios = []*pb.Ref_Scenario{}
		for _, s := range scenarios {
			res.Scenarios = append(res.Scenarios, s.Ref())
		}
	}

	return res, nil
}
