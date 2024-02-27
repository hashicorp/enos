// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"
	"fmt"

	"github.com/hashicorp/enos/internal/diagnostics"
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

	fp, decRes := flightplan.DecodeProto(
		ctx,
		req.GetWorkspace().GetFlightplan(),
		flightplan.DecodeTargetAll,
		req.GetFilter(),
	)
	res.Decode = decRes

	scenarios := fp.Scenarios()
	if len(scenarios) == 0 {
		filter, err := flightplan.NewScenarioFilter(
			flightplan.WithScenarioFilterDecode(req.GetFilter()),
		)
		if err != nil {
			res.Diagnostics = diagnostics.FromErr(err)
		} else {
			res.Diagnostics = diagnostics.FromErr(fmt.Errorf(
				"no scenarios found matching filter '%s'", filter.String(),
			))
		}
	}

	return res, nil
}
