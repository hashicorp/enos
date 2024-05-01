// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenariosValidateConfig shows the flight plan validation response.
func (v *View) ShowScenariosValidateConfig(res *pb.ValidateScenariosConfigurationResponse) error {
	v.WriteDiagnostics(res.GetDiagnostics())
	v.WriteDiagnostics(res.GetDecode().GetDiagnostics())
	v.WriteDiagnostics(res.GetSampleDecode().GetDiagnostics())

	return status.ScenariosValidateConfig(v.settings.GetFailOnWarnings(), res)
}
