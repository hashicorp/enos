package server

import (
	"context"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// GenerateScenarios generates scenario Terraform modules and configuration
func (s *ServiceV1) GenerateScenarios(
	ctx context.Context,
	req *pb.GenerateScenariosRequest,
) (
	*pb.GenerateScenariosResponse,
	error,
) {
	res := &pb.GenerateScenariosResponse{}
	res.Diagnostics, res.Decode, res.Operations = s.dispatch(
		req.GetFilter(),
		&pb.Operation_Request{
			Workspace: req.GetWorkspace(),
			Value:     &pb.Operation_Request_Generate_{},
		},
	)
	return res, nil
}
