package server

import (
	"context"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ValidateScenariosConfiguration validates a flight plan config.
func (s *ServiceV1) ValidateScenariosConfiguration(
	ctx context.Context,
	req *pb.ValidateScenariosConfigurationRequest,
) (
	*pb.ValidateScenariosConfigurationResponse,
	error,
) {
	res := &pb.ValidateScenariosConfigurationResponse{}

	_, decRes := decodeFlightPlan(
		ctx,
		req.GetWorkspace().GetFlightplan(),
		flightplan.DecodeModeFull,
		req.GetFilter(),
	)
	res.Decode = decRes

	return res, nil
}
