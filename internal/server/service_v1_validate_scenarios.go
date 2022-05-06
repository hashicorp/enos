package server

import (
	"context"

	"github.com/hashicorp/enos/internal/execute"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ValidateScenarios generates scenario terraform modules for each scenario
// that has been filtered for the workspace. It then validates the generated
// module and returns the results.
func (s *ServiceV1) ValidateScenarios(
	ctx context.Context,
	req *pb.ValidateScenariosRequest,
) (
	*pb.ValidateScenariosResponse,
	error,
) {
	res := &pb.ValidateScenariosResponse{}
	mods, diags, err := decodeAndGenerate(
		req.GetWorkspace(), req.GetFilter(),
	)
	res.Diagnostics = diags
	if err != nil {
		for _, mod := range mods {
			res.Responses = append(res.Responses, &pb.Scenario_Command_Validate_Response{
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
			).Validate(ctx),
		)
	}

	return res, err
}
