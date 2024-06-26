// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	"github.com/hashicorp/enos/version"
)

// GetVersion returns the version information.
func (s *ServiceV1) GetVersion(
	ctx context.Context,
	req *pb.GetVersionRequest,
) (
	*pb.GetVersionResponse,
	error,
) {
	return &pb.GetVersionResponse{
		Version: version.Version,
		GitSha:  version.GitCommit,
	}, nil
}
