package server

import (
	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func decodeFlightPlan(rfp *pb.FlightPlan) (*flightplan.FlightPlan, []*pb.Diagnostic) {
	dec, err := flightplan.NewDecoder(
		flightplan.WithDecoderBaseDir(rfp.GetBaseDir()),
		flightplan.WithDecoderFPFiles(rfp.GetEnosHcl()),
		flightplan.WithDecoderVarFiles(rfp.GetEnosVarsHcl()),
	)
	if err != nil {
		return nil, diagnostics.FromErr(err)
	}

	diags := dec.Parse()
	if diags.HasErrors() {
		return nil, diagnostics.FromHCL(dec.ParserFiles(), diags)
	}

	fp, moreDiags := dec.Decode()
	diags = diags.Extend(moreDiags)

	return fp, diagnostics.FromHCL(dec.ParserFiles(), diags)
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
