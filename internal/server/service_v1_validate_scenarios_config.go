// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"
	"errors"
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

	if req.GetNoValidateSamples() && req.GetNoValidateScenarios() {
		res.Diagnostics = diagnostics.FromErr(errors.New("cannot validate when given both no_validate_scenarios and no_validate_samples"))
		return res, nil
	}

	var decRes *pb.DecodeResponse
	var fp *flightplan.FlightPlan
	if !req.GetNoValidateScenarios() {
		fp, decRes = flightplan.DecodeProto(
			ctx,
			req.GetWorkspace().GetFlightplan(),
			flightplan.DecodeTargetAll,
			req.GetFilter(),
		)
		res.Decode = decRes
		if diagnostics.HasFailed(
			req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
			res.GetDiagnostics(),
		) {
			return res, nil
		}

		scenarios := fp.Scenarios()
		if len(scenarios) == 0 {
			filter, err := flightplan.NewScenarioFilter(
				flightplan.WithScenarioFilterDecode(req.GetFilter()),
			)
			if err != nil {
				res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
			} else {
				res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(fmt.Errorf(
					"no scenarios found matching filter '%s'", filter.String(),
				))...)
			}
		}
	}

	if !req.GetNoValidateSamples() {
		sampleReq, err := flightplan.NewSampleValidationReq(
			flightplan.WithSampleValidationReqWorkSpace(req.GetWorkspace()),
			flightplan.WithSampleValidationReqFilter(req.GetSampleFilter()),
		)
		if err != nil {
			res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)

			return res, nil
		}

		res.SampleDecode = sampleReq.Validate(ctx)
	}

	return res, nil
}
