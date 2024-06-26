// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// GenerateScenarios generates scenario Terraform modules and configuration.
func (s *ServiceV1) GenerateScenarios(
	ctx context.Context,
	req *pb.GenerateScenariosRequest,
) (
	*pb.GenerateScenariosResponse,
	error,
) {
	res := &pb.GenerateScenariosResponse{}
	res.Diagnostics, res.Decode, res.Operations = s.dispatch(
		ctx,
		req.GetFilter(),
		&pb.Operation_Request{
			Workspace: req.GetWorkspace(),
			Value:     &pb.Operation_Request_Generate_{},
		},
	)

	return res, nil
}
