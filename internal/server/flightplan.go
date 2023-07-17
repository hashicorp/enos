package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func decodeFlightPlan(
	ctx context.Context,
	pfp *pb.FlightPlan,
	mode flightplan.DecodeMode,
	f *pb.Scenario_Filter,
) (*flightplan.FlightPlan, *pb.DecodeResponse) {
	res := &pb.DecodeResponse{
		Diagnostics: []*pb.Diagnostic{},
	}

	opts := []flightplan.DecoderOpt{
		flightplan.WithDecoderBaseDir(pfp.GetBaseDir()),
		flightplan.WithDecoderFPFiles(pfp.GetEnosHcl()),
		flightplan.WithDecoderVarFiles(pfp.GetEnosVarsHcl()),
		flightplan.WithDecoderEnv(pfp.GetEnosVarsEnv()),
		flightplan.WithDecoderDecodeMode(mode),
	}

	sf, err := flightplan.NewScenarioFilter(flightplan.WithScenarioFilterDecode(f))
	if err != nil {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
	} else {
		opts = append(opts, flightplan.WithDecoderScenarioFilter(sf))
	}

	dec, err := flightplan.NewDecoder(opts...)
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

	fp, hclDiags := dec.Decode(ctx)
	if len(hclDiags) > 0 {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromHCL(dec.ParserFiles(), hclDiags)...)
	}

	return fp, res
}
