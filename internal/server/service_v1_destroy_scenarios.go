package server

import (
	"context"

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

	mods, diags, err := decodeAndGetGenRef(
		req.GetWorkspace(),
		req.GetFilter(),
	)

	res.Diagnostics = diags
	if err != nil {
		return res, err
	}

	for _, mod := range mods {
		res.Responses = append(res.Responses,
			execute.NewExecutor(
				execute.WithProtoModuleAndConfig(
					mod.GetTerraformModule(),
					req.GetWorkspace().GetTfExecCfg(),
				),
			).Destroy(ctx),
		)
	}

	return res, nil
}
