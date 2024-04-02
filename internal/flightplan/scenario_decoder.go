// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"cmp"
	"context"
	"fmt"
	"runtime"
	"slices"
	"sync"

	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
)

// ScenarioDecoder decodes filters and decodes scenario blocks to a desired target level.
type ScenarioDecoder struct {
	*hcl.EvalContext
	DecodeTarget
	*ScenarioFilter
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

	if d.DecodeTarget <= DecodeTargetUnset || d.DecodeTarget > DecodeTargetAll {
		return nil, fmt.Errorf(
			"unsupported decode target level: %d, expected a level between %d and %d",
			d.DecodeTarget, DecodeTargetUnset+1, DecodeTargetAll,
		)
	}

	if d.DecodeTarget != DecodeTargetScenariosNamesExpandVariants &&
		d.DecodeTarget != DecodeTargetScenariosMatrixOnly &&
		d.DecodeTarget != DecodeTargetScenariosNamesNoVariants &&
		d.DecodeTarget < DecodeTargetScenariosComplete {
		return nil, fmt.Errorf(
			"unsupported decode target level: %d, expected a level between %d and %d",
			d.DecodeTarget, DecodeTargetUnset+1, DecodeTargetAll,
		)
	}

	return d, nil
}

// DecodedScenarioBlock is a decoded scenario block.
type DecodedScenarioBlock struct {
	Name         string
	Block        *hcl.Block
	DecodeTarget DecodeTarget
	Matrix       *Matrix
	Scenarios    []*Scenario
	Diagnostics  hcl.Diagnostics
}

// DecodedScenarioBlocks are all of the scenario blocks that have been decoded.
type DecodedScenarioBlocks []*DecodedScenarioBlock

func (d DecodedScenarioBlocks) Diagnostics() hcl.Diagnostics {
	if d == nil || len(d) < 1 {
		return nil
	}

	var diags hcl.Diagnostics
	for i := range d {
		diags = append(diags, d[i].Diagnostics...)
	}

	return diags
}

// Scenarios returns all of the scenarios that were decoded.
func (d DecodedScenarioBlocks) Scenarios() []*Scenario {
	if d == nil || len(d) < 1 {
		return nil
	}

	scenarios := []*Scenario{}
	for i := range d {
		scenarios = append(scenarios, d[i].Scenarios...)
	}

	return scenarios
}

// CombinedMatrix returns a combined matrix of all scenario blocks matrices. Uniqueness is by values.
func (d DecodedScenarioBlocks) CombinedMatrix() *Matrix {
	if d == nil || len(d) < 1 {
		return nil
	}

	var m *Matrix
	for i := range d {
		sm := d[i].Matrix
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

// DecodeScenarioBlcoks decodes the "scenario" blocks that are defined in the top-level schem to
// the target level configured in the decode spec.
func (d *ScenarioDecoder) DecodeScenarioBlocks(ctx context.Context, blocks []*hcl.Block) DecodedScenarioBlocks {
	if len(blocks) < 1 {
		return nil
	}

	scenarioBlocks := d.filterScenarioBlocks(blocks)
	for i := range scenarioBlocks {
		// Don't worry about decoding scenario blocks that don't match our name if we've been
		// given a name.
		if d.ScenarioFilter != nil && d.ScenarioFilter.Name != "" {
			if d.ScenarioFilter.Name != scenarioBlocks[i].Name {
				continue
			}
		}

		if d.DecodeTarget >= DecodeTargetScenariosMatrixOnly {
			var diags hcl.Diagnostics
			scenarioBlocks[i].Matrix, diags = decodeMatrix(d.EvalContext, scenarioBlocks[i].Block)
			scenarioBlocks[i].Diagnostics = scenarioBlocks[i].Diagnostics.Extend(diags)

			if scenarioBlocks[i].Matrix != nil &&
				len(scenarioBlocks[i].Matrix.GetVectors()) > 1 &&
				d.ScenarioFilter != nil {
				scenarioBlocks[i].Matrix = scenarioBlocks[i].Matrix.Filter(d.ScenarioFilter)
				if scenarioBlocks[i].Matrix == nil || len(scenarioBlocks[i].Matrix.GetVectors()) < 1 {
					// Our filter has no matches with the scenario filter so there's no need to
					// try and continue to decode.
					continue
				}
			}
		}

		if d.DecodeTarget < DecodeTargetScenariosNamesExpandVariants {
			continue
		}

		// Choose which decode option based on our target and the number of variants we have.
		if scenarioBlocks[i].Matrix == nil ||
			(scenarioBlocks[i].Matrix != nil && len(scenarioBlocks[i].Matrix.GetVectors()) < 1) {
			d.decodeScenariosSerial(scenarioBlocks[i])
		} else {
			switch d.DecodeTarget {
			case DecodeTargetScenariosNamesExpandVariants:
				switch {
				case runtime.NumCPU() < 2:
					d.decodeScenariosSerial(scenarioBlocks[i])
				default:
					d.decodeScenariosConcurrent(ctx, scenarioBlocks[i])
				}
			case DecodeTargetScenariosComplete, DecodeTargetAll:
				switch {
				case runtime.NumCPU() < 2:
					d.decodeScenariosSerial(scenarioBlocks[i])
				default:
					d.decodeScenariosConcurrent(ctx, scenarioBlocks[i])
				}
			default:
				scenarioBlocks[i].Diagnostics = scenarioBlocks[i].Diagnostics.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "unknown scenario decode mode",
					Detail:   fmt.Sprintf("%v is not a known decode mode", d.DecodeTarget),
					Subject:  scenarioBlocks[i].Block.TypeRange.Ptr(),
					Context:  scenarioBlocks[i].Block.DefRange.Ptr(),
				})
			}
		}

		slices.SortStableFunc(scenarioBlocks[i].Scenarios, compareScenarios)
	}

	slices.SortStableFunc(scenarioBlocks, func(a, b *DecodedScenarioBlock) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return scenarioBlocks
}

// compareScenarios takes two scenarios and does a sort comparison. At present we only factor
// in existence, name, and variants into equality.
func compareScenarios(a, b *Scenario) int {
	// Compare by existence
	if a == nil && b == nil {
		return 0
	}

	if a != nil && b == nil {
		return 1
	}

	if a == nil && b != nil {
		return -1
	}

	// Compare by name
	if i := cmp.Compare(a.Name, b.Name); i != 0 {
		return i
	}

	// Compare by variant vectors
	return compareVector(a.Variants, b.Variants)
}

// filterScenarioBlocks takes a slice of hcl.Blocks's and returns our base set of filtered
// DecodedScenarioBlocks.
func (d *ScenarioDecoder) filterScenarioBlocks(blocks []*hcl.Block) DecodedScenarioBlocks {
	if len(blocks) < 1 {
		return nil
	}

	res := DecodedScenarioBlocks{}
	for i := range blocks {
		// If we've got a filter that includes a name and our scenario block doesn't
		// match we don't need to decode anything.
		if d.ScenarioFilter != nil && d.ScenarioFilter.Name != "" && blocks[i].Labels[0] != d.ScenarioFilter.Name {
			continue
		}

		res = append(res, &DecodedScenarioBlock{
			Name:         blocks[i].Labels[0],
			Block:        blocks[i],
			DecodeTarget: d.DecodeTarget,
			Diagnostics:  verifyBlockLabelsAreValidIdentifiers(blocks[i]),
		})
	}

	return res
}

// decodeScenario configures a child eval context and decodes the scenario.
func (d *ScenarioDecoder) decodeScenario(
	vec *Vector,
	block *hcl.Block,
) (bool, *Scenario, hcl.Diagnostics) {
	scenario := NewScenario()
	var diags hcl.Diagnostics

	evalCtx := d.EvalContext.NewChild()
	if vec != nil {
		scenario.Variants = vec
		evalCtx.Variables = map[string]cty.Value{
			"matrix": vec.CtyVal(),
		}
	}

	diags = scenario.decode(block, evalCtx, d.DecodeTarget)

	return !diags.HasErrors(), scenario, diags
}

// decodeScenariosSerial decodes scenario variants serially. When we don't have lots of scenarios
// or we're not fully decoding the scenario this can be a faster option than decoding concurrently
// and requiring the overhead of goroutines.
func (d *ScenarioDecoder) decodeScenariosSerial(sb *DecodedScenarioBlock) {
	// Decode the scenario without a matrix
	if sb.Matrix == nil || len(sb.Matrix.GetVectors()) < 1 {
		keep, scenario, diags := d.decodeScenario(nil, sb.Block)
		sb.Diagnostics = sb.Diagnostics.Extend(diags)
		if keep {
			sb.Scenarios = append(sb.Scenarios, scenario)
		}

		return
	}

	// Decode a scenario for all matrix vectors
	for i := range sb.Matrix.GetVectors() {
		keep, scenario, diags := d.decodeScenario(sb.Matrix.GetVectors()[i], sb.Block)
		sb.Diagnostics = sb.Diagnostics.Extend(diags)
		if keep {
			sb.Scenarios = append(sb.Scenarios, scenario)
		}
	}
}

// decodeScenariosConcurrent decodes scenario variants concurrently. This is for improved speeds
// when fully decoding lots of scenarios.
func (d *ScenarioDecoder) decodeScenariosConcurrent(ctx context.Context, sb *DecodedScenarioBlock) {
	if sb.Matrix == nil || len(sb.Matrix.Vectors) < 1 || runtime.NumCPU() < 2 {
		d.decodeScenariosSerial(sb)

		return
	}

	collectCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	scenarioC := make(chan *Scenario)
	vectorC := make(chan *Vector)
	diagC := make(chan hcl.Diagnostics)
	decodeWg := sync.WaitGroup{}
	workerWg := sync.WaitGroup{}
	scenarios := []*Scenario{}
	diags := hcl.Diagnostics{}
	defer func() {
		close(diagC)
		close(scenarioC)
		close(vectorC)
	}()

	collectDiags := func() {
		for {
			select {
			case diag := <-diagC:
				diags = diags.Extend(diag)

				continue
			default:
			}

			select {
			case <-collectCtx.Done():
				workerWg.Done()
				return
			case diag := <-diagC:
				diags = diags.Extend(diag)
			}
		}
	}
	workerWg.Add(1)
	go collectDiags()

	collectScenarios := func() {
		for {
			select {
			case scenario := <-scenarioC:
				scenarios = append(scenarios, scenario)
				decodeWg.Done()

				continue
			default:
			}

			select {
			case <-collectCtx.Done():
				workerWg.Done()
				return
			case scenario := <-scenarioC:
				scenarios = append(scenarios, scenario)
				decodeWg.Done()
			}
		}
	}
	workerWg.Add(1)
	go collectScenarios()

	decodeScenario := func() {
		for {
			select {
			case vec := <-vectorC:
				keep, scenario, diags := d.decodeScenario(vec, sb.Block)
				diagC <- diags
				if keep {
					scenarioC <- scenario
				} else {
					decodeWg.Done()
				}

				continue
			default:
			}

			select {
			case <-collectCtx.Done():
				workerWg.Done()
				return
			case vec := <-vectorC:
				keep, scenario, diags := d.decodeScenario(vec, sb.Block)
				diagC <- diags
				if keep {
					scenarioC <- scenario
				} else {
					decodeWg.Done()
				}
			}
		}
	}

	for range runtime.NumCPU() {
		workerWg.Add(1)
		go decodeScenario()
	}

	decodeWg.Add(len(sb.Matrix.Vectors))
	for i := range sb.Matrix.Vectors {
		vectorC <- sb.Matrix.Vectors[i]
	}

	decodeWg.Wait()
	cancel()
	workerWg.Wait()
	sb.Scenarios = append(sb.Scenarios, scenarios...)
	sb.Diagnostics = sb.Diagnostics.Extend(diags)
}
