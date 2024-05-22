// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"github.com/hashicorp/enos/internal/ui/status"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// ShowScenariosValidateConfig shows the flight plan validation response.
func (v *View) ShowScenariosValidateConfig(res *pb.ValidateScenariosConfigurationResponse) error {
	v.WriteDiagnostics(res.GetDiagnostics())
	v.WriteDiagnostics(res.GetDecode().GetDiagnostics())
	v.WriteDiagnostics(res.GetSampleDecode().GetDiagnostics())

	return status.ScenariosValidateConfig(v.settings.GetFailOnWarnings(), res)
}
