package server

import (
	"context"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// LaunchScenarios generates scenario terraform modules for each scenario
// that has been filtered for the workspace. It then validates the generated
// module and returns the results.
func (s *ServiceV1) LaunchScenarios(
	ctx context.Context,
	req *pb.LaunchScenariosRequest,
) (
	*pb.LaunchScenariosResponse,
	error,
) {
	res := &pb.LaunchScenariosResponse{
		Responses: []*pb.Scenario_Command_Launch_Response{},
	}

	mods, diags, err := decodeAndGenerate(
		req.GetWorkspace(), req.GetFilter(),
	)
	res.Diagnostics = diags
	if err != nil {
		for _, mod := range mods {
			res.Responses = append(res.Responses, &pb.Scenario_Command_Launch_Response{
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
			).Launch(ctx),
		)
	}

	return res, nil
}
