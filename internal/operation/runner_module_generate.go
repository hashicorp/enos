// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/generate"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// moduleGenerate takes a context, request and event sender and generates a Terraform
// module. Any errors or warnings are returned as diagnostics is the response
// value.
func (r *Runner) moduleGenerate(
	ctx context.Context,
	req *pb.Operation_Request, // for
	events *EventSender,
) *pb.Operation_Response_Generate_ {
	// Set up our generate response value
	resVal := &pb.Operation_Response_Generate_{
		Generate: &pb.Operation_Response_Generate{
			Diagnostics: []*pb.Diagnostic{},
		},
	}

	log := r.log.With(RequestDebugArgs(req)...)

	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		resVal.Generate.Diagnostics = append(resVal.Generate.GetDiagnostics(), diagnostics.FromErr(err)...)
		log.Error("failed to create reference from request", "error", err)

		return resVal
	}

	// Notify that we're running generate
	eventVal := &pb.Operation_Event_Generate{}
	event := newEvent(ref, pb.Operation_STATUS_RUNNING)
	event.Value = eventVal
	if err = events.Publish(event); err != nil {
		resVal.Generate.Diagnostics = append(resVal.Generate.GetDiagnostics(), diagnostics.FromErr(err)...)
		log.Error("failed to send event", "error", err)

		return resVal
	}

	// notifyFail prepares the response for failure and sends a failure
	// event
	notifyFail := func(diags []*pb.Diagnostic) {
		resVal.Generate.Diagnostics = append(resVal.Generate.GetDiagnostics(), diags...)
		event.Status = pb.Operation_STATUS_FAILED
		event.Diagnostics = resVal.Generate.GetDiagnostics()
		eventVal.Generate = resVal.Generate

		if err := events.Publish(event); err != nil {
			resVal.Generate.Diagnostics = append(resVal.Generate.GetDiagnostics(), diagnostics.FromErr(err)...)
			log.Error("failed to send event", "error", err)
		}
	}

	// Make sure our context isn't done before we continue
	select {
	case <-ctx.Done():
		notifyFail(diagnostics.FromErr(ctx.Err()))

		return resVal
	default:
	}

	// Decode our scenario and create our module generator
	gen, scenario, diags := scenarioAndModuleGeneratorForReq(ctx, req)
	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		diags) {
		notifyFail(diags)

		return resVal
	}

	// Generate the module
	err = gen.Generate()
	if err != nil {
		notifyFail(diagnostics.FromErr(err))

		return resVal
	}

	// Prepare our response
	resVal.Generate.TerraformModule = &pb.Terraform_Module{
		ModulePath:  gen.TerraformModulePath(),
		RcPath:      gen.TerraformRCPath(),
		ScenarioRef: scenario.Ref(),
	}

	// Configure our Terraform executor to use the module we generated
	r.TFConfig.WithModule(resVal.Generate.GetTerraformModule())

	// Finalize our responses and event
	event.Status = diagnostics.Status(r.TFConfig.FailOnWarnings, resVal.Generate.GetDiagnostics()...)
	event.Diagnostics = resVal.Generate.GetDiagnostics()
	eventVal.Generate = resVal.Generate

	// Notify that we've finished
	if err := events.Publish(event); err != nil {
		resVal.Generate.Diagnostics = append(resVal.Generate.GetDiagnostics(), diagnostics.FromErr(err)...)
		log.Error("failed to send event", "error", err)
	}
	log.Debug("finished generate")

	return resVal
}

func scenarioAndModuleGeneratorForReq(ctx context.Context, req *pb.Operation_Request) (
	*generate.Generator,
	*flightplan.Scenario,
	[]*pb.Diagnostic,
) {
	filter, err := flightplan.NewScenarioFilter(
		flightplan.WithScenarioFilterFromScenarioRef(req.GetScenario()),
	)
	if err != nil {
		return nil, nil, diagnostics.FromErr(err)
	}

	fp, decRes := flightplan.DecodeProto(
		ctx,
		req.GetWorkspace().GetFlightplan(),
		flightplan.DecodeTargetAll,
		filter.Proto(),
	)

	if diagnostics.HasFailed(
		req.GetWorkspace().GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		return nil, nil, decRes.GetDiagnostics()
	}

	ws := req.GetWorkspace()
	scenarios := fp.Scenarios()
	switch len(scenarios) {
	case 0:
		return nil, nil, diagnostics.FromErr(errors.New("no matching scenarios found"))
	case 1:
	default:
		return nil, nil, diagnostics.FromErr(
			fmt.Errorf("found more than one scenario matching scenario reference: %s",
				scenarios[0].String(),
			))
	}

	baseDir, err := isAbs(ws.GetFlightplan().GetBaseDir())
	if err != nil {
		return nil, nil, diagnostics.FromErr(err)
	}

	// Determine our output directory
	outDir := ws.GetOutDir()
	if outDir == "" {
		outDir = outDirForWorkspace(req.GetWorkspace())
	}

	outDir, err = isAbs(outDir)
	if err != nil {
		return nil, nil, diagnostics.FromErr(err)
	}

	// Generate the module
	gen, err := generate.NewGenerator(
		generate.WithScenario(scenarios[0]),
		generate.WithScenarioBaseDirectory(baseDir),
		generate.WithOutBaseDirectory(outDir),
	)
	if err != nil {
		return nil, nil, diagnostics.FromErr(err)
	}

	return gen, scenarios[0], nil
}

// moduleForReq returns a Terraform module for the request. It does not generate
// it, it can only refer to where it would/should exist.
func moduleForReq(ctx context.Context, req *pb.Operation_Request) (*pb.Terraform_Module, []*pb.Diagnostic) {
	gen, scenario, diags := scenarioAndModuleGeneratorForReq(ctx, req)
	if diagnostics.HasErrors(diags) {
		return nil, diags
	}

	return &pb.Terraform_Module{
		ModulePath:  gen.TerraformModulePath(),
		RcPath:      gen.TerraformRCPath(),
		ScenarioRef: scenario.Ref(),
	}, diags
}
