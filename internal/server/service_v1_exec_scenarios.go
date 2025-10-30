// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// ExecScenarios executes a Terraform sub-command in the context of scenarios.
func (s *ServiceV1) ExecScenarios(
	ctx context.Context,
	req *pb.ExecScenariosRequest,
) (
	*pb.ExecScenariosResponse,
	error,
) {
	res := &pb.ExecScenariosResponse{}
	res.Diagnostics, res.Decode, res.Operations = s.dispatch(
		ctx,
		req.GetFilter(),
		&pb.Operation_Request{
			Workspace: req.GetWorkspace(),
			Value:     &pb.Operation_Request_Exec_{},
		},
	)

	return res, nil
}
