package status

import (
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// GetVersion returns the get version response.
func GetVersion(res *pb.GetVersionResponse) error {
	if HasErrorDiags(res) {
		return Error("unable to get version")
	}

	return nil
}
