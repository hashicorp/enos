package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// RunScenarios generates scenario terraform modules for each scenario
// that has been filtered for the workspace. It then validates the generated
// module, runs it, and then destroys it.
func (s *ServiceV1) RunScenarios(
	ctx context.Context,
	req *pb.RunScenariosRequest,
) (
	*pb.RunScenariosResponse,
	error,
) {
	res := &pb.RunScenariosResponse{
		Responses: []*pb.Scenario_Operation_Run_Response{},
	}

	genRes := decodeAndGenerate(req.GetWorkspace(), req.GetFilter())
	res.Diagnostics = genRes.GetDiagnostics()
	res.Decode = genRes.GetDecode()
	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		res.GetDiagnostics(),
		res.GetDecode().GetDiagnostics(),
	) {
		for _, mod := range genRes.GetResponses() {
			res.Responses = append(res.Responses, &pb.Scenario_Operation_Run_Response{
				Generate: mod,
			})
		}

		return res, nil
	}

	for _, gres := range genRes.GetResponses() {
		runRes := execute.NewExecutor(
			execute.WithProtoModuleAndConfig(
				gres.GetTerraformModule(),
				req.GetWorkspace().GetTfExecCfg(),
			),
		).Run(ctx)
		runRes.Generate = gres
		res.Responses = append(res.Responses, runRes)
	}

	return res, nil
}
