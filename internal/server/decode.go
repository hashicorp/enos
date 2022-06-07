package server

import (
	"path/filepath"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	res := &pb.GenerateScenariosResponse{}

	scenarios, diags := decodeAndFilter(
		ws.GetFlightplan(),
		filter,
	)
	if diagnostics.HasErrors(diags) {
		res.Diagnostics = diags
		return res
	}

	outDir := ws.GetOutDir()
	if outDir == "" {
		outDir = outDirForWorkspace(ws)
	}

	return generateScenarioModules(
		scenarios,
		ws.GetFlightplan().GetBaseDir(),
		outDir,
	)
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
		Responses: []*pb.Scenario_Command_Generate_Response{},
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
		gres := &pb.Scenario_Command_Generate_Response{}
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
		abs, err := filepath.Abs(path)
		if err != nil {
			return abs, status.Errorf(codes.InvalidArgument, err.Error())
		}
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
	[]*pb.Diagnostic,
) {
	fp, diags := decodeFlightPlan(pfp)
	if diagnostics.HasErrors(diags) {
		return nil, diags
	}

	scenarios, diags := filterScenarios(fp, filter)
	if diagnostics.HasErrors(diags) {
		return nil, diags
	}

	return scenarios, diags
}

func decodeAndGetGenRef(
	ws *pb.Workspace,
	f *pb.Scenario_Filter,
) *pb.GenerateScenariosResponse {
	res := &pb.GenerateScenariosResponse{
		Responses: []*pb.Scenario_Command_Generate_Response{},
	}

	scenarios, diags := decodeAndFilter(
		ws.GetFlightplan(),
		f,
	)
	res.Diagnostics = diags
	if diagnostics.HasErrors(diags) || len(scenarios) == 0 {
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
		gres := &pb.Scenario_Command_Generate_Response{}
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
