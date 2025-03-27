// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	hcl "github.com/hashicorp/hcl/v2"
)

// ScenarioDecoder expands and decodes scenario blocks into individual Scenarios.
type ScenarioDecoder struct {
	*hcl.EvalContext
	DecodeTarget
	*ScenarioFilter
	Blocks []*hcl.Block
}

// ScenarioBlock represents a decoded "scenario" block. It, along with a vector from the MatrixBlock,
// can be used to decode and validate individual scenario's into a Scenario.
type ScenarioBlock struct {
	Name         string
	Block        *hcl.Block
	EvalContext  *hcl.EvalContext
	DecodeTarget DecodeTarget
	Scenarios    []*Scenario
	MatrixBlock  *MatrixBlock
}

// ScenarioDecodeRequest is a request to decode a given scenario to the correct target given an
// individual variant Vector, ScenarioBlock and DecodeTarget.
type ScenarioDecodeRequest struct {
	*Vector
	*ScenarioBlock
	DecodeTarget
}

// ScenarioDecodeResponse is a response given from a scenario decoder. It contains a reference to
// the ScenarioDecodeRequest, the decoded Scenario. Any HCL Diagnostics encountered along the way
// are also included.
type ScenarioDecodeResponse struct {
	*Scenario
	hcl.Diagnostics
	*ScenarioDecodeRequest
}

// ScenarioDecoderOpt is a scenario decoder option.
type ScenarioDecoderOpt func(*ScenarioDecoder)

// WithScenarioDecoderEvalContext sets the parent hcl.EvalContext for the decoder.
func WithScenarioDecoderEvalContext(evctx *hcl.EvalContext) func(*ScenarioDecoder) {
	return func(d *ScenarioDecoder) {
		d.EvalContext = evctx
	}
}

// WithScenarioDecoderDecodeTarget sets the desired target level for the scenario decoder.
func WithScenarioDecoderDecodeTarget(t DecodeTarget) func(*ScenarioDecoder) {
	return func(d *ScenarioDecoder) {
		d.DecodeTarget = t
	}
}

// WithScenarioDecoderScenarioFilter sets the decoders scenario filter.
func WithScenarioDecoderScenarioFilter(f *ScenarioFilter) func(*ScenarioDecoder) {
	return func(d *ScenarioDecoder) {
		d.ScenarioFilter = f
	}
}

// WithScenarioDecoderBlocks sets the blocks to decode.
func WithScenarioDecoderBlocks(b []*hcl.Block) func(*ScenarioDecoder) {
	return func(d *ScenarioDecoder) {
		d.Blocks = b
	}
}

// NewScenarioDecoder takes any number of scenario decoder opts and returns a new scenario decoder.
// If the scenario decoder has not been configured in a valid way an error will be returned.
func NewScenarioDecoder(opts ...ScenarioDecoderOpt) (*ScenarioDecoder, error) {
	d := &ScenarioDecoder{
		EvalContext:  &hcl.EvalContext{},
		DecodeTarget: DecodeTargetUnset,
	}

	for i := range opts {
		opts[i](d)
	}

	switch {
	case d.DecodeTarget == DecodeTargetUnset:
		return nil, errors.New("you must provide a decode target level")
	case d.DecodeTarget > DecodeTargetAll:
		return nil, errors.New("invalid decode target")
	default:
	}

	return d, nil
}

// DecodeAll decodes the "scenario" blocks that are defined in the top-level schema to the target
// level configured in the decode spec and adds them to the FlightPlan.
//
// WARNING: Be cautious using this function when dealing with large numbers of scenarios as we'll
// keep a reference in memory for each scenario. If you are performing streaming operations it's
// better to use Iterator() to allow the caller to handle each scenario individually instead of
// generating a huge in-memory slice of Scenarios in the FlightPlan.
func (d *ScenarioDecoder) DecodeAll(ctx context.Context, fp *FlightPlan) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	if d == nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unitialized scenario decoder",
		})
	}

	iter := d.Iterator()
	if iter == nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to create scenario decoder iterator",
		})
	}

	if fp == nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to decode all scenarios into nil flightplan",
		})
	}

	diags = diags.Extend(iter.Start(ctx))
	if diags.HasErrors() {
		return diags
	}
	defer iter.Stop()

	for iter.Next(ctx) {
		diags = diags.Extend(iter.Diagnostics())
		if diags.HasErrors() {
			return diags
		}

		scenarioResponse := iter.Scenario()
		if scenarioResponse == nil || scenarioResponse.Scenario == nil {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "expected scenario",
			})
		}

		block, ok := iter.Blocks().FindByName(scenarioResponse.Scenario.Name)
		if !ok {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("no scenario block with name %s could be found", scenarioResponse.Scenario.Name),
			})
		}

		block.Scenarios = append(block.Scenarios, scenarioResponse.Scenario)
		if scenarioResponse.Diagnostics != nil && scenarioResponse.Diagnostics.HasErrors() {
			return diags.Extend(scenarioResponse.Diagnostics)
		}
	}

	moreDiags := iter.Diagnostics()
	diags = diags.Extend(moreDiags)
	if diags.HasErrors() {
		return diags
	}

	if iter.Count() == 0 {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("no scenarios matched filter criteria: %+v", iter.filter),
		})
	}

	fp.ScenarioBlocks = iter.Blocks().Sort()

	return diags
}

// Iterator return a new ScenarioDecoderIterator that can be used to iteratively decode and return
// scenarios.
func (d *ScenarioDecoder) Iterator() *ScenarioDecoderIterator {
	if d == nil {
		return nil
	}

	return NewScenarioDecoderIterator(d.EvalContext, d.DecodeTarget, d.ScenarioFilter, d.Blocks)
}

// Matrix returns the Scenario matrices Cartesian Product.
func (d *ScenarioBlock) Matrix() *Matrix {
	if d == nil || d.MatrixBlock == nil {
		return nil
	}

	return d.MatrixBlock.Matrix()
}

// ScenarioBlocks are all of the scenario blocks that have been decoded.
type ScenarioBlocks []*ScenarioBlock

func (d ScenarioBlocks) Sort() ScenarioBlocks {
	if d == nil {
		return nil
	}

	slices.SortStableFunc(d, func(a, b *ScenarioBlock) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return d
}

// FindByName finds and returns a reference to a ScenarioBlock by its name label.
func (d ScenarioBlocks) FindByName(name string) (*ScenarioBlock, bool) {
	if d == nil {
		return nil, false
	}

	for i := range d {
		if d[i].Name == name {
			return d[i], true
		}
	}

	return nil, false
}

// Scenarios returns all of the scenarios that were decoded.
func (d ScenarioBlocks) Scenarios() []*Scenario {
	if len(d) < 1 {
		return nil
	}

	scenarios := []*Scenario{}
	for i := range d {
		scenarios = append(scenarios, d[i].Scenarios...)
	}

	return scenarios
}

// CombinedMatrix returns a combined matrix of all scenario blocks matrices. Uniqueness is by values.
func (d ScenarioBlocks) CombinedMatrix() *Matrix {
	if len(d) < 1 {
		return nil
	}

	var m *Matrix
	for i := range d {
		sm := d[i].Matrix()
		if m == nil {
			m = sm
		} else {
			for _, v := range sm.GetVectors() {
				m.AddVector(v)
			}
		}
	}

	if m == nil {
		return nil
	}

	return m.UniqueValues()
}
