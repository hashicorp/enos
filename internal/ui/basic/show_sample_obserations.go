// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"fmt"
	"slices"

	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowSampleObservation shows the sample observation.
func (v *View) ShowSampleObservation(res *pb.ObserveSampleResponse) error {
	header := []string{"sample", "subset", "scenario filter"}
	rows := [][]string{{""}} // add a padding row
	maxAttrs := 0
	for _, elm := range res.GetObservation().GetElements() {
		row := []string{
			elm.GetSample().GetId().GetName(),
			elm.GetSubset().GetId().GetName(),
			elm.GetScenario().GetId().GetFilter(),
		}

		attrs := elm.GetAttributes().AsMap()
		if len(attrs) < 1 {
			rows = append(rows, row)

			continue
		}

		if len(attrs) > maxAttrs {
			maxAttrs = len(attrs)
		}
		keys := []string{}
		for key := range attrs {
			keys = append(keys, key)
		}
		slices.Sort(keys)

		for i := range keys {
			row = append(row, fmt.Sprintf("%s=%v", keys[i], attrs[keys[i]]))
		}

		rows = append(rows, row)
	}

	if maxAttrs > 0 {
		header = append(header, "attributes")
		for i := maxAttrs - 1; i > 0; i-- {
			header = append(header, "")
		}
	}

	if len(rows) > 1 {
		v.ui.RenderTable(header, rows)
	}
	v.WriteDiagnostics(res.GetDecode().GetDiagnostics())
	v.WriteDiagnostics(res.GetDiagnostics())

	return status.ShowSampleObservation(v.settings.GetFailOnWarnings(), res)
}
