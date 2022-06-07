package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// OutputScenarios returns scenario outputs
func (s *ServiceV1) OutputScenarios(
	ctx context.Context,
	req *pb.OutputScenariosRequest,
) (
	*pb.OutputScenariosResponse,
	error,
) {
	res := &pb.OutputScenariosResponse{
		Responses: []*pb.Scenario_Operation_Output_Response{},
	}

	genRef := decodeAndGetGenRef(req.GetWorkspace(), req.GetFilter())
	res.Diagnostics = genRef.GetDiagnostics()
	res.Decode = genRef.GetDecode()
	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		res.GetDiagnostics(),
		res.GetDecode().GetDiagnostics(),
	) {
		return res, nil
	}

	for _, gref := range genRef.GetResponses() {
		outputRes := execute.NewExecutor(
			execute.WithProtoModuleAndConfig(
				gref.GetTerraformModule(),
				req.GetWorkspace().GetTfExecCfg(),
			),
		).Output(ctx)
		outputRes.TerraformModule = gref.GetTerraformModule()
		res.Responses = append(res.Responses, outputRes)
	}

	return res, nil
}
