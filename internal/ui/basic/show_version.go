// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"fmt"

	"github.com/hashicorp/enos/internal/ui/status"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// ShowVersion shows the version information.
func (v *View) ShowVersion(all bool, res *pb.GetVersionResponse) error {
	if !all {
		v.ui.Output(res.GetVersion())
	} else {
		v.ui.Output(fmt.Sprintf("Enos version: %s sha: %s", res.GetVersion(), res.GetGitSha()))
	}

	return status.GetVersion(res)
}
