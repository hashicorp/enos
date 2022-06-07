package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ExecScenarios executes a Terraform sub-command in the context of scenarios
func (s *ServiceV1) ExecScenarios(
	ctx context.Context,
	req *pb.ExecScenariosRequest,
) (
	*pb.ExecScenariosResponse,
	error,
) {
	res := &pb.ExecScenariosResponse{
		Responses: []*pb.Scenario_Operation_Exec_Response{},
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
		execRes := execute.NewExecutor(
			execute.WithProtoModuleAndConfig(
				gref.GetTerraformModule(),
				req.GetWorkspace().GetTfExecCfg(),
			),
		).Exec(ctx)
		execRes.TerraformModule = gref.GetTerraformModule()
		res.Responses = append(res.Responses, execRes)
	}

	return res, nil
}
