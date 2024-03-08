// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	hcl "github.com/hashicorp/hcl/v2"
)

var sampleSubsetSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "attributes", Required: false},
		{Name: "scenario_name", Required: false},
		{Name: "scenario_filter", Required: false},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: blockTypeMatrix},
	},
}

// SampleSubset is a subset of samples.
type SampleSubset struct {
	Name           string
	SampleName     string
	ScenarioName   string
	ScenarioFilter string
	Attributes     cty.Value
	Matrix         *Matrix
}

// NewSampleSubset returns a new SampleSubset.
func NewSampleSubset() *SampleSubset {
	return &SampleSubset{}
}

// Frame takes a context and workspace and decodes and filters the samples scenario matrix into a
// frame.
func (s *SampleSubset) Frame(ctx context.Context, ws *pb.Workspace) (*SampleSubsetFrame, *pb.DecodeResponse) {
	if s == nil {
		return nil, &pb.DecodeResponse{
			Diagnostics: diagnostics.FromErr(errors.New("cannot get frame from nil subset")),
		}
	}

	if ws == nil {
		return nil, &pb.DecodeResponse{
			Diagnostics: diagnostics.FromErr(errors.New("cannot get frame from nil workspace")),
		}
	}

	// Create a scenario filter from our sample subset and decode a flightplan with the intersection
	// of our filter and the actual scenario variants.
	sf, err := NewScenarioFilter(WithScenarioFilterFromSampleSubset(s))
	if err != nil {
		return nil, &pb.DecodeResponse{
			Diagnostics: diagnostics.FromErr(err),
		}
	}

	// Decode our flightplan to matrix level using our filter from our subset. This should result
	// in a combined matrix that we can use as the frame matrix.
	fp, decRes := DecodeProto(
		ctx, ws.GetFlightplan(), DecodeTargetScenariosMatrixOnly, sf.Proto(),
	)
	if diagnostics.HasFailed(ws.GetTfExecCfg().GetFailOnWarnings(), decRes.GetDiagnostics()) {
		return nil, decRes
	}

	// We we didn't find any scenarios matching the filter we don't have a frame.
	if fp == nil || fp.ScenarioBlocks == nil || len(fp.ScenarioBlocks) < 1 {
		return nil, nil
	}

	// Make sure we only found one scenario block with our filter.
	if len(fp.ScenarioBlocks) > 1 {
		found := []string{}
		for i := range fp.ScenarioBlocks {
			found = append(found, fp.ScenarioBlocks[i].Name)
		}

		return nil, &pb.DecodeResponse{
			Diagnostics: diagnostics.FromErr(fmt.Errorf(
				"unsupported sample filter: sample filter expected on scenario %s, found scenarios %s",
				sf.Name, strings.Join(found, " "),
			)),
		}
	}

	return &SampleSubsetFrame{
		SampleSubset:   s,
		ScenarioFilter: sf.Proto(),
		Matrix:         fp.ScenarioBlocks[0].Matrix,
	}, nil
}

// decode takes a sample subset HCL block and decodes and unmarshals the contents of it into itself.
func (s *SampleSubset) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	s.Name = block.Labels[0]

	content, moreDiags := block.Body.Content(sampleSubsetSchema)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	s.ScenarioName, moreDiags = decodeSampleSubsetFieldString("scenario_name", content.Attributes, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	scenarioNameAttr, ok := content.Attributes["scenario_name"]
	if s.ScenarioName != "" && ok {
		moreDiags = verifyValidIdentifier(s.ScenarioName, scenarioNameAttr.NameRange.Ptr())
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			return diags
		}
	}

	s.ScenarioFilter, moreDiags = decodeSampleSubsetFieldString("scenario_filter", content.Attributes, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	attributesAttr, ok := content.Attributes["attributes"]
	if ok {
		s.Attributes, moreDiags = decodeAndValidateSampleAttrs(attributesAttr, ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			return diags
		}
	}

	// Decode the matrix block if there is one.
	s.Matrix, moreDiags = decodeMatrix(ctx, block)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	if s.Name == "" && s.ScenarioName == "" && s.ScenarioFilter == "" {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "A subset name, scenario_name, or scenario_filter is required but not defined",
			Subject:  block.Body.MissingItemRange().Ptr(),
			Context:  block.TypeRange.Ptr(),
		})
	}

	if s.ScenarioFilter != "" && s.Matrix != nil && len(s.Matrix.Vectors) > 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "cannot filter scenarios from subset because the subset has beed configured with both a matrix and scenario filter",
			Subject:  block.Body.MissingItemRange().Ptr(),
			Context:  block.TypeRange.Ptr(),
		})
	}

	if s.ScenarioName != "" && s.ScenarioFilter != "" {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "cannot filter scenarios from subset because a scenario_name and scenario_filter are both defined",
			Subject:  block.Body.MissingItemRange().Ptr(),
			Context:  block.TypeRange.Ptr(),
		})
	}

	return diags
}

func decodeSampleSubsetFieldString(name string, attrs hcl.Attributes, ctx *hcl.EvalContext) (string, hcl.Diagnostics) {
	diags := hcl.Diagnostics{}
	f, ok := attrs[name]
	if !ok {
		return "", nil
	}

	val, moreDiags := f.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return "", diags
	}

	if val.IsNull() {
		return "", diags
	}

	if !val.IsWhollyKnown() {
		return "", diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("value of %s must be knowable", name),
			Subject:  f.NameRange.Ptr(),
			Context:  f.Range.Ptr(),
		})
	}

	if !val.Type().Equals(cty.String) {
		return "", diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("value of %s must be a string, got %s", name, val.Type().GoString()),
			Subject:  f.NameRange.Ptr(),
			Context:  f.Range.Ptr(),
		})
	}

	return val.AsString(), diags
}
