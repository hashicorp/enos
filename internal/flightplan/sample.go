// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"context"
	"errors"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	hcl "github.com/hashicorp/hcl/v2"
)

var sampleSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "attributes", Required: false},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: blockTypeSampleSubset, LabelNames: []string{attrLabelNameDefault}},
	},
}

// A sample is named collection of subsets and attributes.
type Sample struct {
	Name       string
	Attributes cty.Value
	Subsets    []*SampleSubset
}

// NewSample returns a new Sample.
func NewSample() *Sample {
	return &Sample{}
}

// Ref returns the proto reference.
func (s *Sample) Ref() *pb.Ref_Sample {
	return &pb.Ref_Sample{
		Id: &pb.Sample_ID{
			Name: s.Name,
		},
	}
}

// Decode decodes a sample from an HCL block and eval context.
func (s *Sample) Decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	s.Name = block.Labels[0]

	content, moreDiags := block.Body.Content(sampleSchema)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	s.Attributes, moreDiags = decodeAndValidateSampleAttrs(content.Attributes["attributes"], ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	subsets := content.Blocks.OfType(blockTypeSampleSubset)
	if len(subsets) < 1 {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "sample does not contain any defined subsets",
			Detail:   "a sample must contain one-or-more subsets",
			Subject:  block.TypeRange.Ptr(),
			Context:  block.DefRange.Ptr(),
		})
	}

	names := map[string]struct{}{}
	for i := range subsets {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(subsets[i])
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		ss := &SampleSubset{}
		moreDiags = ss.decode(subsets[i], ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			return diags
		}

		if _, ok := names[ss.Name]; ok {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "a subset with the same name has already been declared",
				Detail:   fmt.Sprintf("A subset with the name %s has already been defined", ss.Name),
				Subject:  subsets[i].DefRange.Ptr(),
				Context:  subsets[i].TypeRange.Ptr(),
			})
		}

		names[ss.Name] = struct{}{}
		s.Subsets = append(s.Subsets, ss)
	}

	return diags
}

// Frame takes a context, workspace, and sample filter and decodes and filters a matching field.
func (s *Sample) Frame(
	ctx context.Context,
	ws *pb.Workspace,
	filter *pb.Sample_Filter,
) (
	*SampleFrame,
	*pb.DecodeResponse,
) {
	subsets := s.filterSubsets(filter)
	if len(subsets) < 1 {
		return nil, &pb.DecodeResponse{
			Diagnostics: diagnostics.FromErr(fmt.Errorf("no subsets matched the given filter: %s", filter.String())),
		}
	}

	f := &SampleFrame{
		Sample: s,
		Filter: filter,
	}
	for i := range subsets {
		if i == 0 {
			f.SubsetFrames = SampleSubsetFrames{}
		}
		frame, err := subsets[i].Frame(ctx, ws)
		if err != nil {
			return nil, err
		}

		if frame == nil {
			continue
		}

		frame.SampleName = s.Name
		f.SubsetFrames[subsets[i].Name] = frame
	}

	return f, nil
}

func (s *Sample) filterSubsets(filter *pb.Sample_Filter) []*SampleSubset {
	if s == nil || len(s.Subsets) < 1 {
		return nil
	}

	subsets := s.Subsets
	if f := filter; f != nil {
		if inclSubs := f.GetSubsets(); inclSubs != nil {
			newSubs := []*SampleSubset{}
			for i := range inclSubs {
				for j := range subsets {
					if inclSubs[i].GetName() == subsets[j].Name {
						newSubs = append(newSubs, subsets[j])
						break
					}
				}
			}
			subsets = newSubs
		}

		if exclSubs := f.GetExcludeSubsets(); exclSubs != nil {
			for i := range exclSubs {
				for j := range subsets {
					if exclSubs[i].GetName() == subsets[j].Name {
						subsets = append(subsets[:j], subsets[j+1:]...)
						break
					}
				}
			}
		}
	}

	return subsets
}

func decodeAndValidateSampleAttrs(attr *hcl.Attribute, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	diags := hcl.Diagnostics{}
	if attr == nil {
		return cty.NilVal, nil
	}

	val, moreDiags := attr.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return val, diags
	}

	if val.IsNull() {
		return val, diags
	}

	if !val.IsWhollyKnown() {
		return cty.UnknownVal(val.Type()), diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "sample attributes must be knowable",
			Detail:   "all sample attributes must be knowable when decoding samples",
			Subject:  attr.NameRange.Ptr(),
			Context:  attr.Range.Ptr(),
		})
	}

	if !val.CanIterateElements() {
		return val, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "sample attributes must be an object or map",
			Detail:   "cannot iterate elements of type " + val.Type().GoString(),
			Subject:  attr.NameRange.Ptr(),
			Context:  attr.Range.Ptr(),
		})
	}

	return val, diags
}

// decodeSamples decodes the samples from the flightplan.
func decodeSamples(
	ctx context.Context,
	ws *pb.Workspace,
	filter *pb.Sample_Filter,
) (*FlightPlan, *pb.DecodeResponse) {
	decRes := &pb.DecodeResponse{}

	if ws == nil {
		decRes.Diagnostics = diagnostics.FromErr(errors.New("cannot sample without a configured workspace"))

		return nil, decRes
	}

	if filter == nil {
		decRes.Diagnostics = diagnostics.FromErr(errors.New("cannot sample without a configured filter"))

		return nil, decRes
	}

	efp := ws.GetFlightplan()
	if efp == nil {
		decRes.Diagnostics = diagnostics.FromErr(errors.New("cannot sample without a configured flightplan"))

		return nil, decRes
	}

	// Try and locate the sample we're trying to observe.
	sampleFP, decRes := DecodeProto(
		ctx,
		ws.GetFlightplan(),
		DecodeTargetSamples,
		nil,
	)
	if diagnostics.HasFailed(
		ws.GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		return nil, decRes
	}

	return sampleFP, decRes
}

func findSampleByRef(fp *FlightPlan, ref *pb.Ref_Sample) (*Sample, []*pb.Diagnostic) {
	if fp == nil {
		return nil, diagnostics.FromErr(errors.New("cannot find samples in non-existent FlightPlan"))
	}

	if len(fp.Samples) < 1 {
		return nil, diagnostics.FromErr(errors.New("no samples found"))
	}

	if ref == nil || ref.GetId().GetName() == "" {
		return nil, diagnostics.FromErr(errors.New("no sample name was included in the filter"))
	}

	for i := range fp.Samples {
		if fp.Samples[i].Ref().GetId().GetName() == ref.GetId().GetName() {
			return fp.Samples[i], nil
		}
	}

	return nil, diagnostics.FromErr(fmt.Errorf("no sample found with name %s", ref.GetId().GetName()))
}

// decodeAndFindSample decodes the samples from the flightplan and selects the sample from the filter.
func decodeAndFindSample(
	ctx context.Context,
	ws *pb.Workspace,
	filter *pb.Sample_Filter,
) (*Sample, *pb.DecodeResponse) {
	fp, decRes := decodeSamples(ctx, ws, filter)
	if diagnostics.HasFailed(
		ws.GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		return nil, decRes
	}

	sample, diags := findSampleByRef(fp, filter.GetSample())
	decRes.Diagnostics = append(decRes.GetDiagnostics(), diags...)

	return sample, decRes
}
