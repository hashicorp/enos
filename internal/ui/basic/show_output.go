// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

func (v *View) ShowOutput(res *pb.OperationResponses) error {
	for _, r := range res.GetResponses() {
		v.writeOutputResponse(r)
	}

	err := v.ShowDiagnostics(res.GetDiagnostics())
	if err != nil {
		return err
	}

	return status.OperationResponses(v.Settings().GetFailOnWarnings(), res)
}
