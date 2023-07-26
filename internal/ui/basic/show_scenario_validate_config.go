package basic

import (
	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenariosValidateConfig shows the flight plan validation response.
func (v *View) ShowScenariosValidateConfig(res *pb.ValidateScenariosConfigurationResponse) error {
	diags := diagnostics.Concat(res.GetDecode().GetDiagnostics(), res.GetDiagnostics())
	v.WriteDiagnostics(diags)

	return status.ScenariosValidateConfig(v.settings.GetFailOnWarnings(), res)
}
