package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
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
		Responses: []*pb.Scenario_Operation_Launch_Response{},
	}

	genRes := decodeAndGenerate(req.GetWorkspace(), req.GetFilter())
	res.Diagnostics = genRes.GetDiagnostics()
	res.Decode = genRes.GetDecode()
	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		res.GetDiagnostics(),
		res.GetDecode().GetDiagnostics(),
	) {
		for _, gres := range genRes.GetResponses() {
			res.Responses = append(res.Responses, &pb.Scenario_Operation_Launch_Response{
				Generate: gres,
			})
		}

		return res, nil
	}

	for _, gres := range genRes.GetResponses() {
		launchRes := execute.NewExecutor(
			execute.WithProtoModuleAndConfig(
				gres.GetTerraformModule(),
				req.GetWorkspace().GetTfExecCfg(),
			),
		).Launch(ctx)
		launchRes.Generate = gres
		res.Responses = append(res.Responses, launchRes)
	}

	return res, nil
}
