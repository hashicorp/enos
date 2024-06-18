// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/zclconf/go-cty/cty"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	hcl "github.com/hashicorp/hcl/v2"
)

type matrixDecoder struct{}

// MatrixBlock represent a full "matrix" block at various stages.
type MatrixBlock struct {
	Original        *Matrix
	IncludeProducts []*Matrix
	Excludes        []*Exclude
	FinalProduct    *Matrix
}

func newMatrixDecoder() *matrixDecoder {
	return &matrixDecoder{}
}

func (d *MatrixBlock) Matrix() *Matrix {
	if d == nil {
		return nil
	}

	return d.FinalProduct
}

func (d *MatrixBlock) GetOriginal() *Matrix {
	if d == nil {
		return nil
	}

	return d.Original
}

func (d *MatrixBlock) GetIncludeProducts() []*Matrix {
	if d == nil {
		return nil
	}

	return d.IncludeProducts
}

func (d *MatrixBlock) GetExcludes() []*Exclude {
	if d == nil {
		return nil
	}

	return d.Excludes
}

func (d *MatrixBlock) Filter(f *ScenarioFilter) *Matrix {
	if d == nil || d.FinalProduct == nil {
		return nil
	}

	d.FinalProduct = d.FinalProduct.Filter(f)
	d.FinalProduct.Sort()

	return d.FinalProduct
}

func (d *MatrixBlock) Sort() *Matrix {
	if d == nil || d.FinalProduct == nil {
		return nil
	}

	d.FinalProduct.Sort()

	return d.FinalProduct
}

func (d *MatrixBlock) Set(m *Matrix) {
	if d == nil {
		return
	}

	d.FinalProduct = m
}

// decodeMatrix takes an eval context and scenario blocks and decodes only the matrix block.
// As the matrix block can contain variants, includes, and excludes, the response will contain
// the original variants as a Matrix with those vectors, the decoded includes, decoded excludes
// and the final cartesian product of unique values.
func (md *matrixDecoder) decodeMatrix(
	ctx *hcl.EvalContext,
	block *hcl.Block,
) (*MatrixBlock, hcl.Diagnostics) {
	diags := hcl.Diagnostics{}
	mContent, _, moreDiags := block.Body.PartialContent(matrixSchema)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return nil, diags
	}

	mBlocks := mContent.Blocks.OfType(blockTypeMatrix)
	switch len(mBlocks) {
	case 0:
		// We have no matrix block defined
		return nil, diags
	case 1:
		// Continue
		break
	default:
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "scenario has more than one matrix block defined",
			Detail:   fmt.Sprintf("a single matrix block can be set, found %d", len(mBlocks)),
			Subject:  block.TypeRange.Ptr(),
			Context:  block.DefRange.Ptr(),
		})
	}

	// Let's decode our matrix block into a matrix
	block = mBlocks[0]

	// We'll use our own copy of the eval context so that we can be sure our 'matrix' eval context
	// doesn't leak out.
	evalCtx := ctx.NewChild()
	evalCtx.Variables = ctx.Variables
	evalCtx.Functions = ctx.Functions

	// Each attribute in the matrix should be a variant name whose value must
	// be a list of strings. Convert the value into a matrix vector and add it.
	matrix, moreDiags := md.decodeAndVerifyMatrixBlock(evalCtx, block.Body, false)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return nil, diags
	}
	res := &MatrixBlock{Original: matrix}

	// Now that we have our basic variant vectors in our matrix, we need to combine
	// all vectors into a product that matches all possible unique value combinations.
	res.FinalProduct = res.Original.CartesianProduct().UniqueValues()

	// Now we need to go through all of our blocks and process include and exclude
	// directives. Since HCL allows us to use ordering we'll apply them in the
	// order in which they're defined.
	blockC, remain, moreDiags := block.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: blockTypeMatrixInclude},
			{Type: blockTypeMatrixExclude},
		},
	})
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return nil, diags
	}
	diags = diags.Extend(verifyBodyOnlyHasBlocksWithLabels(
		remain, blockTypeMatrixInclude, blockTypeMatrixExclude,
	))

	for _, mBlock := range blockC.Blocks {
		switch mBlock.Type {
		case "include":
			iMatrix, moreDiags := md.decodeAndVerifyMatrixBlock(evalCtx, mBlock.Body, true)
			diags = diags.Extend(moreDiags)
			if moreDiags != nil && moreDiags.HasErrors() {
				continue
			}

			// Generate our possible include vectors and add them to our main matrix.
			iMatrixProduct := iMatrix.CartesianProduct().UniqueValues()
			res.IncludeProducts = append(res.IncludeProducts, iMatrixProduct)

			for _, vec := range iMatrixProduct.GetVectors() {
				res.FinalProduct.AddVectorSorted(vec)
			}
		case "exclude":
			eMatrix, moreDiags := md.decodeAndVerifyMatrixBlock(evalCtx, mBlock.Body, true)
			diags = diags.Extend(moreDiags)
			if moreDiags != nil && moreDiags.HasErrors() {
				continue
			}

			excludes := []*Exclude{}
			for _, vec := range eMatrix.CartesianProduct().UniqueValues().GetVectors() {
				ex, err := NewExclude(pb.Matrix_Exclude_MODE_CONTAINS, vec)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "unable to generate exclusion filter",
						Detail:   err.Error(),
						Subject:  hcl.RangeBetween(mBlock.LabelRanges[0], mBlock.LabelRanges[1]).Ptr(),
						Context:  mBlock.DefRange.Ptr(),
					})

					continue
				}
				excludes = append(excludes, ex)
			}
			res.Excludes = append(res.Excludes, excludes...)

			// Update our matrix to a copy which has vectors which match our exclusions
			res.FinalProduct = res.FinalProduct.Exclude(excludes...)
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "invalid block in matrix",
				Detail:   "blocks of type include and exclude are supported in matrix blocks, found " + mBlock.Type,
				Subject:  mBlock.TypeRange.Ptr(),
				Context:  mBlock.DefRange.Ptr(),
			})

			continue
		}
	}

	// Return our matrix but do one final pass removing any duplicates that might
	// have been introduced during our inclusions.
	res.FinalProduct = res.FinalProduct.UniqueValues()
	res.FinalProduct.Sort()

	return res, diags
}

// decodeAndVerifyMatrixBlock takes an HCL EvalContext, an HCL Block, and a boolean that determines
// whether or not the block must include attributes only. It then decodes the blocks attributes as
// if they are matrix vectors and returns a new matrix and any diagnostics. Only the initial
// decoding is performed. Additional sub-block, includes, excludes, products, etc. are up to the
// caller. Also note that in attr only blocks the 'matrix' eval context will not be created or
// overwritten from the caller eval context.
func (md *matrixDecoder) decodeAndVerifyMatrixBlock(
	ctx *hcl.EvalContext,
	body hcl.Body,
	attrOnlyBlock bool,
) (*Matrix, hcl.Diagnostics) {
	nm := NewMatrix()
	diags := hcl.Diagnostics{}

	attrs, moreDiags := body.JustAttributes()
	if attrOnlyBlock {
		// JustAttributes() will return an error if there are any blocks in the schema. If we are not
		// decoding an attr only schema we'll ignore diagnostics from this block.
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			return nil, diags
		}
	}

	// Make the values of each attribute available in the eval context as we decode. This allows
	// subsequent values to refer to prior values.
	variants := map[string]cty.Value{}
	if !attrOnlyBlock {
		// We might be decoding blocks so we'll create a new matrix eval context with our attributes.
		if ctx.Variables == nil {
			ctx.Variables = map[string]cty.Value{}
		}
	}
	vecs := map[string]*Vector{}
	for _, attr := range md.sortAttributesByStartByte(attrs) {
		val, vec, moreDiags := md.decodeMatrixAttribute(ctx, attr)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		vecs[attr.Name] = vec

		if !attrOnlyBlock {
			// Update our attr eval context if necessary
			variants[attr.Name] = val
			ctx.Variables["matrix"] = cty.ObjectVal(variants)
		}
	}

	// Make sure sort the variants by name
	names := []string{}
	for name := range vecs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		nm.AddVector(vecs[name])
	}

	return nm, diags
}

// Go maps are intentionally unordered. We need to sort our attributes by start byte so that we can
// continue evaluate them in the order in which they were defined as that will allow us to populate
// attribute values in the eval context as we do for variables, locals, and globals.
func (md *matrixDecoder) sortAttributesByStartByte(attrs map[string]*hcl.Attribute) []*hcl.Attribute {
	sorted := []*hcl.Attribute{}
	for _, attr := range attrs {
		sorted = append(sorted, attr)
	}
	slices.SortStableFunc(sorted, func(a, b *hcl.Attribute) int {
		return cmp.Compare(a.Range.Start.Byte, b.Range.Start.Byte)
	})

	return sorted
}

func (md *matrixDecoder) decodeMatrixAttribute(
	ctx *hcl.EvalContext,
	attr *hcl.Attribute,
) (cty.Value, *Vector, hcl.Diagnostics) {
	diags := hcl.Diagnostics{}
	vec := NewVector()

	val, moreDiags := attr.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return val, vec, diags
	}

	if !val.CanIterateElements() {
		return val, vec, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "matrix attribute value must be a list of strings",
			Detail:   fmt.Sprintf("expected value for %s to be a list of strings, found %s", attr.Name, val.Type().GoString()),
			Subject:  attr.NameRange.Ptr(),
			Context:  attr.Range.Ptr(),
		})
	}

	if len(val.AsValueSlice()) == 0 {
		return val, vec, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "matrix attribute values cannot be empty lists",
			Subject:  attr.NameRange.Ptr(),
			Context:  attr.Range.Ptr(),
		})
	}

	for _, elm := range val.AsValueSlice() {
		if !elm.Type().Equals(cty.String) {
			return val, vec, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "matrix attribute value must be a list of strings",
				Detail:   "found element with type " + elm.GoString(),
				Subject:  attr.NameRange.Ptr(),
				Context:  attr.Range.Ptr(),
			})
		}

		vec.Add(NewElement(attr.Name, elm.AsString()))
	}

	return val, vec, diags
}
