// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package status

import (
	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// GetVersion returns the get version response.
func GetVersion(res *pb.GetVersionResponse) error {
	if HasErrorDiags(res) {
		return Error("unable to get version")
	}

	return nil
}
