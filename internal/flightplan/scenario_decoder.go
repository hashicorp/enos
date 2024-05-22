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

	if d.DecodeTarget != DecodeTargetScenariosOutlines &&
		d.DecodeTarget != DecodeTargetScenariosNamesExpandVariants &&
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
	Name            string
	Block           *hcl.Block
	EvalContext     *hcl.EvalContext
	DecodeTarget    DecodeTarget
	Scenarios       []*Scenario
	Diagnostics     hcl.Diagnostics
	DecodedMatrices *DecodedMatrices
}

func (d *DecodedScenarioBlock) Matrix() *Matrix {
	if d == nil || d.DecodedMatrices == nil {
		return nil
	}

	return d.DecodedMatrices.Matrix()
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

// decodeMatrix matrix takes in a scenario block and decodes the embedded matrix block in the scenario.
// This is a prerequisite for decoding all scenario/variant combinations for a scenario block.
func (d *ScenarioDecoder) decodeMatrix(block *DecodedScenarioBlock) {
	if d == nil {
		return
	}

	var diags hcl.Diagnostics
	block.DecodedMatrices, diags = decodeMatrix(d.EvalContext, block.Block)
	block.Diagnostics = block.Diagnostics.Extend(diags)

	// Maybe filter
	if block.Matrix() != nil && len(block.Matrix().GetVectors()) > 1 && d.ScenarioFilter != nil {
		block.DecodedMatrices.Set(block.DecodedMatrices.Filter(d.ScenarioFilter))
	}
}

// decodeScenarioOutline is a special decoding target that only decodes a single instance of the
// a scenario block, which we can use to formulate the overall outline of a scenario. It should
// not be used for scenario operations other than outlining.
func (d *ScenarioDecoder) decodeScenarioOutline(sb *DecodedScenarioBlock) {
	if d == nil || sb == nil {
		return
	}

	var vec *Vector
	m := sb.Matrix()
	if m != nil {
		if vecs := m.GetVectors(); len(vecs) > 0 {
			vec = vecs[0]
		}
	}

	oldTargetLevel := d.DecodeTarget
	defer func() {
		d.DecodeTarget = oldTargetLevel
	}()
	d.DecodeTarget = DecodeTargetAll
	keep, scenario, diags := d.DecodeScenario(vec, sb.Block)
	sb.Diagnostics = sb.Diagnostics.Extend(diags)
	if keep {
		sb.Scenarios = append(sb.Scenarios, scenario)
	}
}

// DecodeScenarioBlcoks decodes the "scenario" blocks that are defined in the top-level schem to
// the target level configured in the decode spec.
func (d *ScenarioDecoder) DecodeScenarioBlocks(ctx context.Context, blocks []*hcl.Block) DecodedScenarioBlocks {
	if len(blocks) < 1 {
		return nil
	}

	scenarioBlocks := d.filterScenarioBlocks(blocks)
	for _, scenarioBlock := range scenarioBlocks {
		// Don't worry about decoding scenario blocks that don't match our name if we've been
		// given a name.
		if d.ScenarioFilter != nil && d.ScenarioFilter.Name != "" {
			if d.ScenarioFilter.Name != scenarioBlock.Name {
				continue
			}
		}

		if d.DecodeTarget >= DecodeTargetScenariosMatrixOnly {
			d.decodeMatrix(scenarioBlock)
		}

		if d.DecodeTarget == DecodeTargetScenariosOutlines {
			d.decodeScenarioOutline(scenarioBlock)
			continue
		}

		if d.DecodeTarget < DecodeTargetScenariosNamesExpandVariants {
			continue
		}

		// Choose which decode option based on our target and the number of variants we have.
		if scenarioBlock.Matrix() == nil ||
			(scenarioBlock.Matrix() != nil && len(scenarioBlock.Matrix().GetVectors()) < 1) {
			d.decodeScenariosSerial(scenarioBlock)
		} else {
			switch d.DecodeTarget {
			case DecodeTargetScenariosNamesExpandVariants, DecodeTargetScenariosComplete, DecodeTargetAll:
				switch {
				case runtime.NumCPU() < 2:
					d.decodeScenariosSerial(scenarioBlock)
				default:
					d.decodeScenariosConcurrent(ctx, scenarioBlock)
				}
			default:
				scenarioBlock.Diagnostics = scenarioBlock.Diagnostics.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "unknown scenario decode mode",
					Detail:   fmt.Sprintf("%v is not a known decode mode", d.DecodeTarget),
					Subject:  scenarioBlock.Block.TypeRange.Ptr(),
					Context:  scenarioBlock.Block.DefRange.Ptr(),
				})
			}
		}

		slices.SortStableFunc(scenarioBlock.Scenarios, compareScenarios)
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
			EvalContext:  d.EvalContext,
			DecodeTarget: d.DecodeTarget,
			Diagnostics:  verifyBlockLabelsAreValidIdentifiers(blocks[i]),
		})
	}

	return res
}

// DecodeScenario configures a child eval context and decodes the scenario.
func (d *ScenarioDecoder) DecodeScenario(
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
	if sb == nil {
		return
	}

	// Decode the scenario without a matrix
	if sb.Matrix() == nil || len(sb.Matrix().GetVectors()) < 1 {
		keep, scenario, diags := d.DecodeScenario(nil, sb.Block)
		sb.Diagnostics = sb.Diagnostics.Extend(diags)
		if keep {
			sb.Scenarios = append(sb.Scenarios, scenario)
		}

		return
	}

	// Decode a scenario for all matrix vectors
	for i := range sb.Matrix().GetVectors() {
		keep, scenario, diags := d.DecodeScenario(sb.Matrix().GetVectors()[i], sb.Block)
		sb.Diagnostics = sb.Diagnostics.Extend(diags)
		if keep {
			sb.Scenarios = append(sb.Scenarios, scenario)
		}
		if sb.Diagnostics != nil && sb.Diagnostics.HasErrors() {
			return
		}
	}
}

// decodeScenariosConcurrent decodes scenario variants concurrently. This is for improved speeds
// when fully decoding lots of scenarios.
func (d *ScenarioDecoder) decodeScenariosConcurrent(ctx context.Context, sb *DecodedScenarioBlock) {
	if sb.Matrix() == nil || len(sb.Matrix().GetVectors()) < 1 || runtime.NumCPU() < 2 {
		d.decodeScenariosSerial(sb)

		return
	}

	decodeCtx, cancelDecode := context.WithCancel(ctx)
	defer cancelDecode()

	diagCtx, cancelDiag := context.WithCancel(ctx)
	defer cancelDiag()

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
				if diags != nil && diags.HasErrors() {
					cancelDecode()
				}

				continue
			default:
			}

			select {
			case <-diagCtx.Done():
				workerWg.Done()
				return
			case diag := <-diagC:
				diags = diags.Extend(diag)
				if diags != nil && diags.HasErrors() {
					cancelDecode()
				}
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
			case <-decodeCtx.Done():
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
				keep, scenario, diags := d.DecodeScenario(vec, sb.Block)
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
			case <-decodeCtx.Done():
				workerWg.Done()
				return
			case vec := <-vectorC:
				keep, scenario, diags := d.DecodeScenario(vec, sb.Block)
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

OUTER:
	for i := range sb.Matrix().GetVectors() {
		select {
		case <-decodeCtx.Done():
			break OUTER
		default:
			decodeWg.Add(1)
			vectorC <- sb.Matrix().GetVectors()[i]
		}
	}

	decodeWg.Wait()
	cancelDecode()
	cancelDiag()
	workerWg.Wait()
	sb.Scenarios = append(sb.Scenarios, scenarios...)
	sb.Diagnostics = sb.Diagnostics.Extend(diags)
}
