// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/memory"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
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
	var scenarioDecoder *flightplan.ScenarioDecoder
	if !req.GetNoValidateScenarios() {
		_, scenarioDecoder, decRes = flightplan.DecodeProto(
			ctx,
			req.GetWorkspace().GetFlightplan(),
			flightplan.DecodeTargetAll,
			req.GetFilter(),
		)
		res.Decode = decRes

		if diagnostics.HasFailed(
			req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
			decRes.GetDiagnostics(),
		) {
			return res, nil
		}

		if scenarioDecoder == nil {
			decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromErr(errors.New(
				"failed to decode scenarios",
			))...)

			return res, nil
		}

		iter := scenarioDecoder.Iterator()
		if iter == nil {
			decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromErr(errors.New(
				"failed to decode scenarios",
			))...)

			return res, nil
		}

		if moreDiags := iter.Start(ctx); moreDiags.HasErrors() {
			decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromHCL(nil, moreDiags)...)
			return res, nil
		}
		defer iter.Stop()

		for iter.Next(ctx) {
			if moreDiags := iter.Diagnostics(); moreDiags.HasErrors() {
				decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromHCL(nil, moreDiags)...)

				return res, nil
			}

			scenarioResponse := iter.Scenario()
			if scenarioResponse == nil {
				decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromErr(errors.New(
					"unable to retrieve scenario from decoder",
				))...)

				return res, nil
			}

			if moreDiags := scenarioResponse.Diagnostics; moreDiags.HasErrors() {
				decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromHCL(nil, moreDiags)...)

				return res, nil
			}
		}

		if iter.Count() == 0 {
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

			return res, nil
		}

		if moreDiags := iter.Diagnostics(); len(moreDiags) > 0 {
			decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromHCL(nil, moreDiags)...)

			return res, nil
		}
	}

	if !req.GetNoValidateSamples() {
		stat, err := memory.Stat(ctx, memory.WithGC())
		if err != nil {
			decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromErr(err)...)

			return res, nil
		}

		sampleReq, err := flightplan.NewSampleValidationReq(
			// Validating samples can be very memory intensive since we're creating lots of matrix
			// products for each sample and subset frame. Set our worker count to either the number of
			// CPUs, or a worker for 1 GiB of available memory, whichever is lower. Make sure we always
			// provision at least one worker.
			flightplan.WithSampleValidationWorkerCount(
				int(
					math.Max(
						float64(1),
						math.Min(
							float64(runtime.NumCPU()),
							float64(stat.Available())/math.Pow(2, 30)),
					),
				),
			),
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
