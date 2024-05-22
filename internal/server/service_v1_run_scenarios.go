// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// RunScenarios generates scenario terraform modules for each scenario
// that has been filtered for the workspace. It then validates the generated
// module, runs it, and then destroys it.
func (s *ServiceV1) RunScenarios(
	ctx context.Context,
	req *pb.RunScenariosRequest,
) (
	*pb.RunScenariosResponse,
	error,
) {
	res := &pb.RunScenariosResponse{}
	res.Diagnostics, res.Decode, res.Operations = s.dispatch(
		ctx,
		req.GetFilter(),
		&pb.Operation_Request{
			Workspace: req.GetWorkspace(),
			Value:     &pb.Operation_Request_Run_{},
		},
	)

	return res, nil
}
