package server

import (
	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func decodeFlightPlan(pfp *pb.FlightPlan) (*flightplan.FlightPlan, *pb.Scenario_Operation_Decode_Response) {
	res := &pb.Scenario_Operation_Decode_Response{Diagnostics: []*pb.Diagnostic{}}

	dec, err := flightplan.NewDecoder(
		flightplan.WithDecoderBaseDir(pfp.GetBaseDir()),
		flightplan.WithDecoderFPFiles(pfp.GetEnosHcl()),
		flightplan.WithDecoderVarFiles(pfp.GetEnosVarsHcl()),
	)
	if err != nil {
		res.Diagnostics = diagnostics.FromErr(err)
		return nil, res
	}

	hclDiags := dec.Parse()
	if len(hclDiags) > 0 {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromHCL(dec.ParserFiles(), hclDiags)...)
	}

	if diagnostics.HasErrors(res.GetDiagnostics()) {
		return nil, res
	}

	fp, hclDiags := dec.Decode()
	if len(hclDiags) > 0 {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromHCL(dec.ParserFiles(), hclDiags)...)
	}

	return fp, res
}

// filterScenarios takes CLI arguments that may contain a scenario filter and
// returns the filtered results.
func filterScenarios(fp *flightplan.FlightPlan, f *pb.Scenario_Filter) ([]*flightplan.Scenario, []*pb.Diagnostic) {
	filter, err := flightplan.NewScenarioFilter(
		flightplan.WithScenarioFilterDecode(f),
	)
	if err != nil {
		return nil, diagnostics.FromErr(err)
	}

	return fp.ScenariosSelect(filter), nil
}
