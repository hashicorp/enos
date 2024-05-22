// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// LaunchScenarios generates scenario terraform modules for each scenario
// that has been filtered for the workspace. It then validates the generated
// module and returns the results.
func (s *ServiceV1) LaunchScenarios(
	ctx context.Context,
	req *pb.LaunchScenariosRequest,
) (
	*pb.LaunchScenariosResponse,
	error,
) {
	res := &pb.LaunchScenariosResponse{}
	res.Diagnostics, res.Decode, res.Operations = s.dispatch(
		ctx,
		req.GetFilter(),
		&pb.Operation_Request{
			Workspace: req.GetWorkspace(),
			Value:     &pb.Operation_Request_Launch_{},
		},
	)

	return res, nil
}
