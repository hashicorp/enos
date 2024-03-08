// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// Operation takes an operation request with an operation ID and returns
// the value of the operation.
func (s *ServiceV1) Operation(
	ctx context.Context,
	req *pb.OperationRequest,
) (*pb.OperationResponse, error) {
	res, err := s.operator.Response(req.GetOp())

	return &pb.OperationResponse{
		Diagnostics: diagnostics.FromErr(err),
		Response:    res,
	}, nil
}
