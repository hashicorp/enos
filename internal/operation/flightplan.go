package operation

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func isAbs(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return filepath.Abs(path)
	}

	return path, nil
}

func outDirForWorkspace(w *pb.Workspace) string {
	return filepath.Join(w.GetFlightplan().GetBaseDir(), ".enos")
}

func decodeFlightPlan(ctx context.Context, pfp *pb.FlightPlan) (*flightplan.FlightPlan, *pb.DecodeResponse) {
	res := &pb.DecodeResponse{
		Diagnostics: []*pb.Diagnostic{},
	}

	dec, err := flightplan.NewDecoder(
		flightplan.WithDecoderBaseDir(pfp.GetBaseDir()),
		flightplan.WithDecoderFPFiles(pfp.GetEnosHcl()),
		flightplan.WithDecoderVarFiles(pfp.GetEnosVarsHcl()),
		flightplan.WithDecoderEnv(pfp.GetEnosVarsEnv()),
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

	fp, hclDiags := dec.Decode(ctx)
	if len(hclDiags) > 0 {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromHCL(dec.ParserFiles(), hclDiags)...)
	}

	return fp, res
}
