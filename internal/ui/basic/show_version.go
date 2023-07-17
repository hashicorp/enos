package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowVersion shows the version information.
func (v *View) ShowVersion(all bool, res *pb.GetVersionResponse) error {
	if !all {
		v.ui.Output(res.Version)
	} else {
		v.ui.Output(fmt.Sprintf("Enos version: %s sha: %s", res.GetVersion(), res.GetGitSha()))
	}

	return status.GetVersion(res)
}
