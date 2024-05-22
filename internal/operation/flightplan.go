// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"path/filepath"

	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

func isAbs(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return filepath.Abs(path)
	}

	return path, nil
}

func outDirForWorkspace(w *pb.Workspace) string {
	return filepath.Join(w.GetFlightplan().GetBaseDir(), ".enos")
}
