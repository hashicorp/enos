// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// CheckScenarios generates scenario terraform modules for each scenario
// that has been filtered for the workspace. It then validates and plans the
// generated module.
func (s *ServiceV1) CheckScenarios(
	ctx context.Context,
	req *pb.CheckScenariosRequest,
) (
	*pb.CheckScenariosResponse,
	error,
) {
	res := &pb.CheckScenariosResponse{}
	res.Diagnostics, res.Decode, res.Operations = s.dispatch(
		ctx,
		req.GetFilter(),
		&pb.Operation_Request{
			Workspace: req.GetWorkspace(),
			Value:     &pb.Operation_Request_Check_{},
		},
	)

	return res, nil
}
