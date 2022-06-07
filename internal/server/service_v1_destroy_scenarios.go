package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// DestroyScenarios destroys scenarios
func (s *ServiceV1) DestroyScenarios(
	ctx context.Context,
	req *pb.DestroyScenariosRequest,
) (
	*pb.DestroyScenariosResponse,
	error,
) {
	res := &pb.DestroyScenariosResponse{
		Responses: []*pb.Scenario_Command_Destroy_Response{},
	}

	genRef := decodeAndGetGenRef(req.GetWorkspace(), req.GetFilter())
	res.Diagnostics = genRef.GetDiagnostics()
	if diagnostics.HasErrors(res.Diagnostics) {
		return res, nil
	}

	for _, gref := range genRef.GetResponses() {
		destroyRes := execute.NewExecutor(
			execute.WithProtoModuleAndConfig(
				gref.GetTerraformModule(),
				req.GetWorkspace().GetTfExecCfg(),
			),
		).Destroy(ctx)
		destroyRes.TerraformModule = gref.GetTerraformModule()
		res.Responses = append(res.Responses, destroyRes)
	}

	return res, nil
}
