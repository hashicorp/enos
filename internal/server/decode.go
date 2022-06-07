package server

import (
	"path/filepath"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/generate"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// decodeAndGenerate decodes scenarios, filters any out, and generates modules
// for the scenarios.
func decodeAndGenerate(
	ws *pb.Workspace,
	filter *pb.Scenario_Filter,
) *pb.GenerateScenariosResponse {
	scenarios, decRes := decodeAndFilter(
		ws.GetFlightplan(),
		filter,
	)

	if diagnostics.HasFailed(ws.GetTfExecCfg().GetFailOnWarnings(), decRes.GetDiagnostics()) {
		return &pb.GenerateScenariosResponse{
			Decode: decRes,
		}
	}

	outDir := ws.GetOutDir()
	if outDir == "" {
		outDir = outDirForWorkspace(ws)
	}

	modRes := generateScenarioModules(
		scenarios,
		ws.GetFlightplan().GetBaseDir(),
		outDir,
	)
	modRes.Decode = decRes

	return modRes
}

func outDirForWorkspace(w *pb.Workspace) string {
	return filepath.Join(w.Flightplan.GetBaseDir(), ".enos")
}

func generateScenarioModules(
	scenarios []*flightplan.Scenario,
	baseDir string,
	outDir string,
) *pb.GenerateScenariosResponse {
	res := &pb.GenerateScenariosResponse{
		Responses: []*pb.Scenario_Operation_Generate_Response{},
	}

	if len(scenarios) == 0 {
		return res
	}

	baseDir, err := isAbs(baseDir)
	if err != nil {
		res.Diagnostics = diagnostics.FromErr(err)
		return res
	}

	outDir, err = isAbs(outDir)
	if err != nil {
		res.Diagnostics = diagnostics.FromErr(err)
		return res
	}

	for _, scenario := range scenarios {
		gres := &pb.Scenario_Operation_Generate_Response{}
		gen, err := generate.NewGenerator(
			generate.WithScenario(scenario),
			generate.WithScenarioBaseDirectory(baseDir),
			generate.WithOutBaseDirectory(outDir),
		)
		if err != nil {
			gres.Diagnostics = diagnostics.FromErr(err)
			res.Responses = append(res.Responses, gres)
			continue
		}

		err = gen.Generate()
		if err != nil {
			gres.Diagnostics = diagnostics.FromErr(err)
			res.Responses = append(res.Responses, gres)
			continue
		}
		gres.TerraformModule = &pb.Terraform_Module{
			ModulePath:  gen.TerraformModulePath(),
			RcPath:      gen.TerraformRCPath(),
			ScenarioRef: scenario.Ref(),
		}
		res.Responses = append(res.Responses, gres)
	}

	return res
}

func isAbs(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return filepath.Abs(path)
	}

	return path, nil
}

// decodeAndFilter decodes a workspace flightplan into scenarios and filters
// any out.
func decodeAndFilter(
	pfp *pb.FlightPlan,
	filter *pb.Scenario_Filter,
) (
	[]*flightplan.Scenario,
	*pb.Scenario_Operation_Decode_Response,
) {
	fp, decRes := decodeFlightPlan(pfp)
	if diagnostics.HasErrors(decRes.GetDiagnostics()) {
		return nil, decRes
	}

	scenarios, moreDiags := filterScenarios(fp, filter)
	decRes.Diagnostics = append(decRes.GetDiagnostics(), moreDiags...)

	return scenarios, decRes
}

func decodeAndGetGenRef(
	ws *pb.Workspace,
	f *pb.Scenario_Filter,
) *pb.GenerateScenariosResponse {
	res := &pb.GenerateScenariosResponse{
		Responses: []*pb.Scenario_Operation_Generate_Response{},
	}

	scenarios, decRes := decodeAndFilter(
		ws.GetFlightplan(),
		f,
	)
	res.Decode = decRes
	if diagnostics.HasFailed(ws.GetTfExecCfg().GetFailOnWarnings(), decRes.GetDiagnostics()) || len(scenarios) == 0 {
		return res
	}

	outDir := ws.GetOutDir()
	if outDir == "" {
		outDir = outDirForWorkspace(ws)
	}
	outDir, err := isAbs(outDir)
	if err != nil {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
		return res
	}

	baseDir := ws.GetFlightplan().GetBaseDir()
	baseDir, err = isAbs(baseDir)
	if err != nil {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
		return res
	}

	for _, scenario := range scenarios {
		gres := &pb.Scenario_Operation_Generate_Response{}
		gen, err := generate.NewGenerator(
			generate.WithScenario(scenario),
			generate.WithScenarioBaseDirectory(baseDir),
			generate.WithOutBaseDirectory(outDir),
		)
		if err != nil {
			gres.Diagnostics = diagnostics.FromErr(err)
			res.Responses = append(res.Responses, gres)
			continue
		}

		gres.TerraformModule = &pb.Terraform_Module{
			ModulePath:  gen.TerraformModulePath(),
			RcPath:      gen.TerraformRCPath(),
			ScenarioRef: scenario.Ref(),
		}
		res.Responses = append(res.Responses, gres)
	}

	return res
}
