package server

import (
	"context"

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
		Responses: []*pb.Scenario_Command_Exec_Response{},
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
			).Exec(ctx),
		)
	}

	return res, nil
}
