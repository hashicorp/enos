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
) ([]*pb.Scenario_Command_Generate_Response, []*pb.Diagnostic, error) {
	scenarios, diags, err := decodeAndFilter(
		ws.GetFlightplan(),
		filter,
	)
	if err != nil {
		return nil, diags, err
	}

	outDir := ws.GetOutDir()
	if outDir == "" {
		outDir = outDirForWorkspace(ws)
	}

	mods, err := generateScenarioModules(
		scenarios,
		ws.GetFlightplan().GetBaseDir(),
		outDir,
	)

	return mods, diags, err
}

func outDirForWorkspace(w *pb.Workspace) string {
	return filepath.Join(w.Flightplan.GetBaseDir(), ".enos")
}

func generateScenarioModules(
	scenarios []*flightplan.Scenario,
	baseDir string,
	outDir string,
) ([]*pb.Scenario_Command_Generate_Response, error) {
	var err error
	responses := []*pb.Scenario_Command_Generate_Response{}

	if len(scenarios) == 0 {
		return responses, nil
	}

	baseDir, err = isAbs(baseDir)
	if err != nil {
		return responses, err
	}

	outDir, err = isAbs(outDir)
	if err != nil {
		return responses, err
	}

	for _, scenario := range scenarios {
		res := &pb.Scenario_Command_Generate_Response{}
		gen, err := generate.NewGenerator(
			generate.WithScenario(scenario),
			generate.WithScenarioBaseDirectory(baseDir),
			generate.WithOutBaseDirectory(outDir),
		)
		if err != nil {
			res.Diagnostics = diagnostics.FromErr(err)
			responses = append(responses, res)
			continue
		}

		err = gen.Generate()
		if err != nil {
			res.Diagnostics = diagnostics.FromErr(err)
			responses = append(responses, res)
			continue
		}
		res.TerraformModule = &pb.Terraform_Module{
			ModulePath:  gen.TerraformModulePath(),
			RcPath:      gen.TerraformRCPath(),
			ScenarioRef: scenario.Ref(),
		}
		responses = append(responses, res)
	}

	return responses, nil
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
) ([]*flightplan.Scenario, []*pb.Diagnostic, error) {
	fp, diags := decodeFlightPlan(pfp)
	if len(diags) > 0 {
		return nil, diags, status.Error(codes.InvalidArgument, "unable to decode flight plan")
	}

	scenarios, diags := filterScenarios(fp, filter)
	if len(diags) > 0 {
		return nil, diags, status.Error(codes.InvalidArgument, "unable to filter scenarios")
	}

	return scenarios, diags, nil
}

func decodeAndGetGenRef(
	ws *pb.Workspace,
	f *pb.Scenario_Filter,
) (
	[]*pb.Scenario_Command_Generate_Response,
	[]*pb.Diagnostic,
	error,
) {
	scenarios, diags, err := decodeAndFilter(
		ws.GetFlightplan(),
		f,
	)
	if err != nil || len(diags) > 0 || len(scenarios) == 0 {
		return nil, diags, err
	}

	outDir := ws.GetOutDir()
	if outDir == "" {
		outDir = outDirForWorkspace(ws)
	}
	outDir, err = isAbs(outDir)
	if err != nil {
		return nil, diags, status.Errorf(codes.InvalidArgument, "unable to decode flight plan: %s", err.Error())
	}

	baseDir := ws.GetFlightplan().GetBaseDir()
	baseDir, err = isAbs(baseDir)
	if err != nil {
		return nil, diags, status.Errorf(codes.InvalidArgument, "unable to decode flight plan: %s", err.Error())
	}

	responses := []*pb.Scenario_Command_Generate_Response{}

	for _, scenario := range scenarios {
		res := &pb.Scenario_Command_Generate_Response{}
		gen, err := generate.NewGenerator(
			generate.WithScenario(scenario),
			generate.WithScenarioBaseDirectory(baseDir),
			generate.WithOutBaseDirectory(outDir),
		)
		if err != nil {
			res.Diagnostics = diagnostics.FromErr(err)
			responses = append(responses, res)
			continue
		}

		res.TerraformModule = &pb.Terraform_Module{
			ModulePath:  gen.TerraformModulePath(),
			RcPath:      gen.TerraformRCPath(),
			ScenarioRef: scenario.Ref(),
		}
		responses = append(responses, res)
	}

	return responses, nil, nil
}
