// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package status

import (
	"fmt"

	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// Format returns the format status.
func Format(cfg *pb.FormatRequest_Config, res *pb.FormatResponse) error {
	checkFailed := false
	var err error

	for _, out := range res.GetResponses() {
		if cfg.GetCheck() && out.GetChanged() {
			checkFailed = true
		}

		if !HasErrorDiags(out) {
			continue
		}

		err = Error(fmt.Sprintf("formatting %s failed", out.GetPath()), err)
	}

	if HasErrorDiags(res) {
		err = Error("unable to format configuration", err)
	}

	if checkFailed {
		return &ErrExit{
			ExitCode: 3,
			Err:      err,
			Msg:      "check failed",
		}
	}

	return err
}
