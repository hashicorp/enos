package server

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
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

	genRes := decodeAndGenerate(req.GetWorkspace(), req.GetFilter())
	res.Diagnostics = genRes.GetDiagnostics()
	if diagnostics.HasErrors(res.GetDiagnostics()) ||
		(req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings() && diagnostics.HasWarnings(res.GetDiagnostics())) {
		for _, gres := range genRes.GetResponses() {
			res.Responses = append(res.Responses, &pb.Scenario_Command_Validate_Response{
				Generate: gres,
			})
		}

		return res, nil
	}

	for _, gres := range genRes.GetResponses() {
		execRes := execute.NewExecutor(
			execute.WithProtoModuleAndConfig(
				gres.GetTerraformModule(),
				req.GetWorkspace().GetTfExecCfg(),
			),
		).Validate(ctx)
		execRes.Generate = gres
		res.Responses = append(res.Responses, execRes)
	}

	return res, nil
}
