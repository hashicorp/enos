// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// DestroyScenarios destroys scenarios.
func (s *ServiceV1) DestroyScenarios(
	ctx context.Context,
	req *pb.DestroyScenariosRequest,
) (
	*pb.DestroyScenariosResponse,
	error,
) {
	res := &pb.DestroyScenariosResponse{}
	res.Diagnostics, res.Decode, res.Operations = s.dispatch(
		ctx,
		req.GetFilter(),
		&pb.Operation_Request{
			Workspace: req.GetWorkspace(),
			Value:     &pb.Operation_Request_Destroy_{},
		},
	)

	return res, nil
}
