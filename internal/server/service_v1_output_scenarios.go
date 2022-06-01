package server

import (
	"context"

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
		Responses: []*pb.Scenario_Command_Output_Response{},
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
			).Output(ctx),
		)
	}

	return res, nil
}
