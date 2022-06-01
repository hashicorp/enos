package server

import (
	"context"

	"github.com/hashicorp/enos/internal/version"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// GetVersion returns the version information
func (s *ServiceV1) GetVersion(
	ctx context.Context,
	req *pb.GetVersionRequest,
) (
	*pb.GetVersionResponse,
	error,
) {
	return &pb.GetVersionResponse{
		Version: version.Version,
		GitSha:  version.GitSHA,
	}, nil
}
