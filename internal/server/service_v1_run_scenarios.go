package server

import (
	"context"

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
		Responses: []*pb.Scenario_Command_Run_Response{},
	}

	mods, diags, err := decodeAndGenerate(
		req.GetWorkspace(), req.GetFilter(),
	)
	res.Diagnostics = diags
	if err != nil {
		for _, mod := range mods {
			res.Responses = append(res.Responses, &pb.Scenario_Command_Run_Response{
				Generate: mod,
			})
		}

		return res, err
	}

	for _, mod := range mods {
		res.Responses = append(res.Responses,
			execute.NewExecutor(
				execute.WithProtoModuleAndConfig(
					mod.GetTerraformModule(),
					req.GetWorkspace().GetTfExecCfg(),
				),
			).Run(ctx),
		)
	}

	return res, nil
}
