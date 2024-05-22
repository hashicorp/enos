// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
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
