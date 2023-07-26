package server

import (
	"context"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// OutputScenarios returns scenario outputs.
func (s *ServiceV1) OutputScenarios(
	ctx context.Context,
	req *pb.OutputScenariosRequest,
) (
	*pb.OutputScenariosResponse,
	error,
) {
	res := &pb.OutputScenariosResponse{}
	res.Diagnostics, res.Decode, res.Operations = s.dispatch(
		ctx,
		req.GetFilter(),
		&pb.Operation_Request{
			Workspace: req.GetWorkspace(),
			Value:     &pb.Operation_Request_Output_{},
		},
	)

	return res, nil
}
