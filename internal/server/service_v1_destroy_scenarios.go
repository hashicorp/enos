package server

import (
	"context"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// DestroyScenarios destroys scenarios
func (s *ServiceV1) DestroyScenarios(
	ctx context.Context,
	req *pb.DestroyScenariosRequest,
) (
	*pb.DestroyScenariosResponse,
	error,
) {
	res := &pb.DestroyScenariosResponse{}
	res.Diagnostics, res.Decode, res.Operations = s.dispatch(
		req.GetFilter(),
		&pb.Operation_Request{
			Workspace: req.GetWorkspace(),
			Value:     &pb.Operation_Request_Destroy_{},
		},
	)
	return res, nil
}
