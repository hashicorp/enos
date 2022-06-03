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
	return decodeAndGenerate(req.GetWorkspace(), req.GetFilter()), nil
}
