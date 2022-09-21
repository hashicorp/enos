package server

import (
	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// decodeAndFilter decodes a workspace flightplan into scenarios and filters
// any out.
func decodeAndFilter(
	pfp *pb.FlightPlan,
	filter *pb.Scenario_Filter,
) (
	[]*flightplan.Scenario,
	*pb.DecodeResponse,
) {
	fp, res := decodeFlightPlan(pfp)
	if diagnostics.HasErrors(res.GetDiagnostics()) {
		return nil, res
	}

	// Return filtering diagnostics as part of decoding failure for now
	scenarios, moreDiags := filterScenarios(fp, filter)
	res.Diagnostics = append(res.GetDiagnostics(), moreDiags...)

	return scenarios, res
}
