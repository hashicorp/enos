// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowSampleList shows the a list of samples.
func (v *View) ShowSampleList(res *pb.ListSamplesResponse) error {
	header := []string{"name"}
	rows := [][]string{{""}} // add a padding row
	for _, ref := range res.GetSamples() {
		rows = append(rows, []string{ref.GetId().GetName()})
	}

	if len(rows) > 1 {
		v.ui.RenderTable(header, rows)
	}
	v.WriteDiagnostics(res.GetDecode().GetDiagnostics())
	v.WriteDiagnostics(res.GetDiagnostics())

	return status.ListSamples(v.settings.GetFailOnWarnings(), res)
}
