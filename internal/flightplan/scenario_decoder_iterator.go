// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
)

// ScenarioDecoderIterator is an iteratable struct that decodes scenarios and allows the caller to
// choose how to handle the decoded scenarios.
type ScenarioDecoderIterator struct {
	blocks         []*hcl.Block
	evalCtx        *hcl.EvalContext
	filter         *ScenarioFilter
	decodeTarget   DecodeTarget
	diags          hcl.Diagnostics
	scenarioBlocks ScenarioBlocks
	nextScenario   *ScenarioDecodeResponse
	nextScenarioC  chan *ScenarioDecodeResponse
	decodeWorkerWg sync.WaitGroup
	hasStarted     bool
	hasFailed      bool
	mustDispatch   int
	haveReturned   int
	cancel         func()
}

func NewScenarioDecoderIterator(
	evalCtx *hcl.EvalContext,
	decodeTarget DecodeTarget,
	filter *ScenarioFilter,
	blocks []*hcl.Block,
) *ScenarioDecoderIterator {
	return &ScenarioDecoderIterator{
		evalCtx:      evalCtx,
		decodeTarget: decodeTarget,
		filter:       filter,
		blocks:       blocks,
	}
}

// Start starts the scenario decoder iterator. Use Next() to determine if there's another scenario
// available. Use `Scenario()` to retrieve the value after calling Next(). Use Diagnostics() to
// retrieve any error diagnostics. Always call Stop() when completed to clean-up resources. Use
// Done() to determine if the interator has finished decoding and returning all scenarios.
func (d *ScenarioDecoderIterator) Start(ctx context.Context) hcl.Diagnostics {
	if d == nil {
		diags := hcl.Diagnostics{}
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "scenario decode was not correctly initialized",
		})
	}

	if d.diags == nil {
		d.diags = hcl.Diagnostics{}
	}

	if d.Done() {
		return d.diags.Append(&hcl.Diagnostic{
			Summary: "completed scenario decoder cannot be started",
		})
	}

	if d.blocks == nil {
		return d.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "no scenarios were found",
		})
	}

	d.hasStarted = true
	d.nextScenarioC = make(chan *ScenarioDecodeResponse)
	d.decodeWorkerWg = sync.WaitGroup{}
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = sync.OnceFunc(cancel)
	d.diags = d.diags.Extend(d.startDecoder(ctx))
	if d.diags.HasErrors() {
		d.hasFailed = true
	}

	return d.diags
}

// Done presents whether or not the iterator has completed all available work.
func (d *ScenarioDecoderIterator) Done() bool {
	if d == nil || !d.hasStarted {
		return false
	}

	if d.hasFailed {
		return true
	}

	return d.mustDispatch == d.haveReturned
}

// Next determines if there are any more scenarios to decode and selects one to be retrieved by
// Scenario(). Any error diagnostics can be retrieved with Diagnostics().
func (d *ScenarioDecoderIterator) Next(ctx context.Context) bool {
	if d == nil {
		return false
	}

	if d.Done() {
		return false
	}

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			d.diags = d.diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "timed out waiting for next scenario: " + err.Error(),
			})
		}
		d.hasFailed = true
		d.cancel()

		return false
	default:
	}

	// Our timer here should never happen but we inject an intentional short circuit here so that
	// we dont wait for the context deadline if something unexpected goes wrong.
	timeout := time.NewTimer(2 * time.Second)
	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			d.diags = d.diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "timed out waiting for next scenario: " + err.Error(),
			})
		}
		d.hasFailed = true
		d.cancel()

		return false
	case <-timeout.C:
		d.diags = d.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "timed out waiting for next scenario",
		})
		d.hasFailed = true
		d.cancel()

		return false
	case res := <-d.nextScenarioC:
		d.nextScenario = res
		d.haveReturned++
		if res != nil {
			if res.Diagnostics != nil {
				if res.Diagnostics.HasErrors() {
					d.diags = d.diags.Extend(res.Diagnostics)
					d.hasFailed = true
					d.cancel()
				}
			}
		}

		return true
	}
}

// Scenario returns the ScenarioDecodeResponse that was queued via Next().
func (d *ScenarioDecoderIterator) Scenario() *ScenarioDecodeResponse {
	if d == nil {
		return nil
	}

	return d.nextScenario
}

// Error returns any error diagnostics encountered while decoding the last ScenarioDecodeResponse.
func (d *ScenarioDecoderIterator) Diagnostics() hcl.Diagnostics {
	if d == nil {
		diags := hcl.Diagnostics{}

		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "scenario decoder has not been correctly initialized",
		})
	}

	return d.diags
}

// Count returns how many scenarios have been decoded and returned by the iterator.
func (d *ScenarioDecoderIterator) Count() int {
	if d == nil {
		return 0
	}

	return d.haveReturned
}

// Blocks returns the ScenarioBlocks that the iterator has been configured wit. Must be called
// after Start() otherwise no blocks will have been decoded.
func (d *ScenarioDecoderIterator) Blocks() ScenarioBlocks {
	if d == nil {
		return nil
	}

	return d.scenarioBlocks
}

// decodeScenarioBlock filters out any unnecessary scenario blocks, sorts them, and then decodes
// and expands any matrix blocks in the scenario blocks.
func (d *ScenarioDecoderIterator) decodeScenarioBlocks(ctx context.Context) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	if len(d.blocks) < 1 {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "no scenario blocks have been defined",
		})
	}

	// Filter out any scenarios we don't need to worry about
	moreDiags := d.filterHCLBlocks()
	if (moreDiags != nil && moreDiags.HasErrors()) || len(d.scenarioBlocks) < 1 {
		return diags.Extend(moreDiags)
	}

	// Sort them so that our outputs are mostly deterministic
	slices.SortStableFunc(d.scenarioBlocks, func(a, b *ScenarioBlock) int {
		return cmp.Compare(a.Name, b.Name)
	})

	moreDiags = d.decodeScenarioBlocksMatrix(ctx)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags.Extend(moreDiags)
	}

	return diags
}

// decodeScenarioBlocksMatrix decodes the matrix block for each scenario block. We do this in
// parallel.
func (d *ScenarioDecoderIterator) decodeScenarioBlocksMatrix(ctx context.Context) hcl.Diagnostics {
	diags := hcl.Diagnostics{}
	diagC := make(chan hcl.Diagnostics)
	diagCloseC := make(chan struct{})
	wgDiag := sync.WaitGroup{}
	wgDecode := sync.WaitGroup{}

	// Start the diagnostics collector
	wgDiag.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case moreDiags := <-diagC:
				diags = diags.Extend(moreDiags)
			default:
			}

			select {
			case <-ctx.Done():
				return
			case <-diagCloseC:
				return
			case moreDiags := <-diagC:
				diags = diags.Extend(moreDiags)
			}
		}
	})

	// Decode any matrix blocks embedded in the scenario blocks. The resulting
	// elements in the scenario blocks matrix product will only include those that
	// intersect with the filter product (if defined).
	for _, scenarioBlock := range d.scenarioBlocks {
		wgDecode.Go(func() {
			d.decodeMatrix(ctx, scenarioBlock, diagC)
		})
	}

	// Wait for the parallel matrix decode and filters and collect any diagnostics.
	wgDecode.Wait()
	close(diagCloseC)
	wgDiag.Wait()

	if diags.HasErrors() {
		return diags
	}

	// Perform our final filtering pass while we're decoding. At this point we
	// should have already excluded scenario blocks which don't match our filter
	// name (if specified, in filterHCLBlocks()), and we'll only have a matrix
	// that intersects with our filter (if specified, decodeMatrix()). We still
	// have to account for filters that don't include a scenario name and none
	// of the scenarios matrix variants match (or there are no matrix variants
	// at all.)
	d.filterScenarioBlocksWithMatrixBlocks()

	// Calculate our expected number of scenario(s)
	for _, scenarioBlock := range d.scenarioBlocks {
		if scenarioBlock.Matrix() == nil {
			d.mustDispatch++
		} else {
			d.mustDispatch += len(scenarioBlock.Matrix().GetVectors())
		}
	}

	return diags
}

// filterScenarioBlocksWithMatrixBlocks filters the scenario blocks after the
// the matrix decode happens. At this point we should have already filtered out
// blocks that don't match our scenario name (if specified) and decoded only
// matrix variants that match the filter (if specified). What we have yet to
// account for is filters that do not include a name but include variants. If we
// have either a scenario block with either no matrix block or no matching
// vectors we should skip it.
func (d *ScenarioDecoderIterator) filterScenarioBlocksWithMatrixBlocks() {
	n := 0
	for _, scenarioBlock := range d.scenarioBlocks {
		if d.filter != nil && d.filter.RequiresVariants() && (scenarioBlock.Matrix() == nil || len(scenarioBlock.Matrix().GetVectors()) < 1) {
			continue
		}
		d.scenarioBlocks[n] = scenarioBlock
		n++
	}
	d.scenarioBlocks = d.scenarioBlocks[:n]
}

// startDecoder starts a scenario decoder which decodes the scenario blocks, expands and filters
// the scenario block matrices, and publishes decoded scenarios to the response channel.
func (d *ScenarioDecoderIterator) startDecoder(ctx context.Context) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	// Decode our blocks by expanding our scenario matrices and applying any filters.
	diags = diags.Extend(d.decodeScenarioBlocks(ctx))
	if diags.HasErrors() {
		return diags
	}

	// Now decode individual Scenarios to the correct target level
	switch d.decodeTarget {
	case DecodeTargetScenariosMatrixOnly:
		// We're actually already done at this point because we've expanded the matrix when decoding
		// the scenario blocks. Signal to our iterator that we're done by setting our haveReturned
		// variable to equal all that we expected to dispatch.
		d.haveReturned = d.mustDispatch
	case DecodeTargetScenariosOutlines:
		// Outlines are a special case we handle individually.
		d.startDecodingScenarioOutlines(ctx)
	case
		// Decode to all other levels that are known.
		DecodeTargetScenariosNamesExpandVariants,
		DecodeTargetScenariosComplete,
		DecodeTargetAll:

		// Choose our decoding worker strategy depending on the machine that we have and the expected
		// number of scenarios. It's not worth the concurrency machinery on weak machines or when we
		// only have a few scenarios blocks to decode.
		switch {
		case d.mustDispatch < 100, runtime.NumCPU() < 2:
			d.decodeWorkerWg.Add(1)
			go d.startDecodingScenariosSerially(ctx, d.decodeTarget)
		default:
			d.decodeWorkerWg.Add(1)
			go d.startDecodingScenariosConcurrently(ctx, d.decodeTarget)
		}
	default:
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("%v is not a known decode mode", d.decodeTarget),
		})
	}

	return diags
}

// Stop cancels any running decode workers and waits for them to be done and then cleans up shared
// communication resources.
func (d *ScenarioDecoderIterator) Stop() {
	// Cancel our worker context so that all of our child goroutines exit
	d.cancel()
	// Wait for all workers to exit
	d.decodeWorkerWg.Wait()
	// Close our channels
	close(d.nextScenarioC)
}

// decodeMatrix matrix takes in a scenario block and decodes the embedded matrix block in the
// scenario. This is a prerequisite for decoding all scenario/variant combinations for a scenario
// block.
func (d *ScenarioDecoderIterator) decodeMatrix(
	ctx context.Context,
	block *ScenarioBlock,
	diagC chan hcl.Diagnostics,
) {
	if d == nil {
		return
	}

	var moreDiags hcl.Diagnostics
	block.MatrixBlock, moreDiags = decodeMatrix(d.evalCtx.NewChild(), block.Block)
	if moreDiags.HasErrors() {
		select {
		case <-ctx.Done():
		case diagC <- moreDiags:
		}
	}

	if block.Matrix() != nil && len(block.Matrix().GetVectors()) > 1 {
		if d.filter != nil {
			// Filter if we've been given one.
			block.MatrixBlock.Filter(d.filter)
		}

		// Always sort our matrix so that we give deterministic results on small sets. The nature of
		// our streaming decoding does not guarantee ordering.
		block.MatrixBlock.Sort()
	}
}

// filterHCLBlocks takes a slice of hcl.Blocks's and creates our initial collection of
// scenarioBlocks that we'll decode.
func (d *ScenarioDecoderIterator) filterHCLBlocks() hcl.Diagnostics {
	if d == nil || len(d.blocks) < 1 {
		return hcl.Diagnostics{}
	}

	d.scenarioBlocks = ScenarioBlocks{}
	for i := range d.blocks {
		// If we've got a filter that includes a name and our scenario block doesn't
		// match we don't need to decode anything.
		if d.filter != nil && d.filter.Name != "" && d.blocks[i].Labels[0] != d.filter.Name {
			continue
		}

		moreDiags := verifyBlockLabelsAreValidIdentifiers(d.blocks[i])
		if moreDiags.HasErrors() {
			return moreDiags
		}

		moreDiags = verifyBlockHasNLabels(d.blocks[i], 1)
		if moreDiags.HasErrors() {
			return moreDiags
		}

		d.scenarioBlocks = append(d.scenarioBlocks, &ScenarioBlock{
			Name:         d.blocks[i].Labels[0],
			Block:        d.blocks[i],
			EvalContext:  d.evalCtx,
			DecodeTarget: d.decodeTarget,
		})
	}

	return nil
}

// startDecodingScenarioOutlines is a special decoding target that only decodes a single instance
// of the a scenario block. We can use one instance to formulate the overall outline of a scenario.
// It should not be used for scenario operations other than outlining.
func (d *ScenarioDecoderIterator) startDecodingScenarioOutlines(
	ctx context.Context,
) {
	if d == nil {
		return
	}

	d.mustDispatch = len(d.scenarioBlocks)
	for i := range d.scenarioBlocks {
		m := d.scenarioBlocks[i].Matrix()
		if m != nil {
			if vecs := m.GetVectors(); len(vecs) > 0 {
				nm := NewMatrix()
				nm.AddVector(vecs[0])
				d.scenarioBlocks[i].MatrixBlock.Set(nm)
			}
		}
	}

	d.decodeWorkerWg.Add(1)
	go d.startDecodingScenariosSerially(ctx, DecodeTargetAll)
}

// startDecodingScenariosSerially decodes scenario variants serially. When we don't have lots of
// scenarios or we're not fully decoding the scenario this can be a faster option than decoding
// concurrently and requiring the overhead of goroutines and message passing through channels.
func (d *ScenarioDecoderIterator) startDecodingScenariosSerially(
	ctx context.Context,
	decodeTarget DecodeTarget,
) {
	if d == nil {
		return
	}

	defer d.decodeWorkerWg.Done()

	if len(d.scenarioBlocks) < 1 {
		return
	}

	for _, sb := range d.scenarioBlocks {
		// Decode the scenario without a matrix
		if sb.Matrix() == nil || len(sb.Matrix().GetVectors()) < 1 {
			select {
			case <-ctx.Done():
				return
			default:
			}

			res := decodeScenario(&ScenarioDecodeRequest{
				Vector:        nil,
				ScenarioBlock: sb,
				DecodeTarget:  decodeTarget,
			})
			select {
			case <-ctx.Done():
				return
			case d.nextScenarioC <- res:
			}

			continue
		}

		// Decode a scenario for all matrix vectors
		for i := range sb.Matrix().GetVectors() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			res := decodeScenario(&ScenarioDecodeRequest{
				Vector:        sb.Matrix().GetVectors()[i],
				ScenarioBlock: sb,
				DecodeTarget:  decodeTarget,
			})

			select {
			case <-ctx.Done():
				return
			case d.nextScenarioC <- res:
			}
		}
	}
}

// startDecodingScenariosConcurrently decodes scenario variants concurrently. This is for improved
// speeds when fully decoding many scenarios on a machine with multiple cores.
func (d *ScenarioDecoderIterator) startDecodingScenariosConcurrently(
	ctx context.Context,
	decodeTarget DecodeTarget,
) {
	if d == nil || d.scenarioBlocks == nil || len(d.scenarioBlocks) < 1 {
		return
	}
	defer d.decodeWorkerWg.Done()

	reqC := make(chan *ScenarioDecodeRequest)

	decodeScenarioWorker := func(ctx context.Context) {
		defer d.decodeWorkerWg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-reqC:
				res := decodeScenario(req)
				select {
				case <-ctx.Done():
					return
				case d.nextScenarioC <- res:
				}
			}
		}
	}

	// Start decode workers
	for range int(math.Max(float64(runtime.NumCPU()), float64(2))) {
		d.decodeWorkerWg.Add(1)
		go decodeScenarioWorker(ctx)
	}

	// Start decode producers
	for _, sb := range d.scenarioBlocks {
		// Decode the scenario without a matrix
		if sb.Matrix() == nil || len(sb.Matrix().GetVectors()) < 1 {
			select {
			case <-ctx.Done():
				return
			default:
			}

			select {
			case <-ctx.Done():
				return
			case reqC <- &ScenarioDecodeRequest{
				Vector:        nil,
				ScenarioBlock: sb,
				DecodeTarget:  decodeTarget,
			}:
			}

			continue
		}

		// Decode a scenario for all matrix vectors
		for i := range sb.Matrix().GetVectors() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			select {
			case <-ctx.Done():
				return
			case reqC <- &ScenarioDecodeRequest{
				Vector:        sb.Matrix().GetVectors()[i],
				ScenarioBlock: sb,
				DecodeTarget:  decodeTarget,
			}:
			}
		}
	}
}

// decodeScenario configures a child eval context and decodes the scenario.
func decodeScenario(req *ScenarioDecodeRequest) *ScenarioDecodeResponse {
	res := &ScenarioDecodeResponse{
		Scenario:              NewScenario(),
		ScenarioDecodeRequest: req,
	}

	evalCtx := req.ScenarioBlock.EvalContext.NewChild()
	if req.Vector != nil {
		res.Scenario.Variants = req.Vector
		evalCtx.Variables = map[string]cty.Value{
			"matrix": req.Vector.CtyVal(),
		}
	}

	res.Diagnostics = res.Scenario.decode(req.ScenarioBlock.Block, evalCtx, req.DecodeTarget)

	return res
}
