// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// ShowFormat displays the response of a format request.
func (v *View) ShowFormat(cfg *pb.FormatRequest_Config, res *pb.FormatResponse) error {
	for _, r := range res.GetResponses() {
		if cfg.GetList() && r.GetChanged() && r.GetPath() != "STDIN" {
			v.ui.Output(r.GetPath())
		}

		if r.GetChanged() {
			if diff := r.GetDiff(); diff != "" {
				v.ui.Output(diff)
			} else if r.GetPath() == "STDIN" && r.GetBody() != "" {
				v.ui.Output(r.GetBody())
			}
		}

		v.WriteDiagnostics(r.GetDiagnostics())
	}

	v.WriteDiagnostics(res.GetDiagnostics())

	return status.Format(cfg, res)
}
