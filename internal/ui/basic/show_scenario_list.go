package basic

import (
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowScenarioList shows the a list of scenarios
func (v *View) ShowScenarioList(res *pb.ListScenariosResponse) error {
	header := []string{"name"}
	rows := [][]string{{""}} // add a padding row
	for i, ref := range res.GetScenarios() {
		scenario := flightplan.NewScenario()
		scenario.FromRef(ref)

		if i == 0 && scenario.Variants != nil {
			for vi := range scenario.Variants {
				if vi == 0 {
					header = append(header, "variants")
					continue
				}
				// create a blank "header" for every variant
				header = append(header, "")
			}
		}

		row := []string{scenario.Name}
		if scenario.Variants != nil {
			for _, elm := range scenario.Variants {
				row = append(row, elm.String())
			}
			rows = append(rows, row)
		}
	}

	if len(rows) > 1 {
		v.ui.RenderTable(header, rows)
	}
	v.WriteDiagnostics(res.GetDecode().GetDiagnostics())
	v.WriteDiagnostics(res.GetDiagnostics())

	return status.ListScenarios(v.settings.GetFailOnWarnings(), res)
}
