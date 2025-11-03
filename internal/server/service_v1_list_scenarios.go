// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"errors"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	"github.com/hashicorp/hcl/v2"
)

// ListScenarios returns a list of scenarios and their variants.
func (s *ServiceV1) ListScenarios(req *pb.ListScenariosRequest, stream pb.EnosService_ListScenariosServer) error {
	diags := hcl.Diagnostics{}

	_, scenarioDecoder, decRes := flightplan.DecodeProto(
		stream.Context(),
		req.GetWorkspace().GetFlightplan(),
		flightplan.DecodeTargetScenariosNamesExpandVariants,
		req.GetFilter(),
	)

	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		return sendListScenarioDecodeResponse(req, stream, decRes, nil)
	}

	if scenarioDecoder == nil {
		return sendListScenarioDecodeResponse(req, stream, decRes, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "failed to decode scenarios",
		}))
	}

	iter := scenarioDecoder.Iterator()
	if iter == nil {
		return sendListScenarioDecodeResponse(req, stream, decRes, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "failed to decode scenarios",
		}))
	}

	if moreDiags := iter.Start(stream.Context()); moreDiags != nil && moreDiags.HasErrors() {
		return sendListScenarioDecodeResponse(req, stream, decRes, moreDiags)
	}
	defer iter.Stop()

	for iter.Next(stream.Context()) {
		if moreDiags := iter.Diagnostics(); moreDiags != nil && moreDiags.HasErrors() {
			return sendListScenarioDecodeResponse(req, stream, decRes, moreDiags)
		}

		scenarioResponse := iter.Scenario()
		if scenarioResponse == nil {
			return sendListScenarioDecodeResponse(req, stream, decRes, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unable to retrieve scenario from decoder",
			}))
		}

		if moreDiags := scenarioResponse.Diagnostics; moreDiags != nil && moreDiags.HasErrors() {
			return sendListScenarioDecodeResponse(req, stream, decRes, moreDiags)
		}

		err := stream.Send(&pb.EnosServiceListScenariosResponse{
			Response: &pb.EnosServiceListScenariosResponse_Scenario{
				Scenario: scenarioResponse.Scenario.Ref(),
			},
		})
		if err != nil {
			return err
		}
	}

	return sendListScenarioDecodeResponse(req, stream, decRes, iter.Diagnostics())
}

func sendListScenarioDecodeResponse(
	req *pb.ListScenariosRequest,
	stream pb.EnosService_ListScenariosServer,
	decRes *pb.DecodeResponse,
	diags hcl.Diagnostics,
) error {
	if decRes == nil {
		return errors.New("no decode response was initialized")
	}

	if len(diags) > 0 {
		decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromHCL(nil, diags)...)
	}

	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		err := stream.Send(&pb.EnosServiceListScenariosResponse{
			Response: &pb.EnosServiceListScenariosResponse_Decode{
				Decode: decRes,
			},
		})

		return err
	}

	return nil
}
