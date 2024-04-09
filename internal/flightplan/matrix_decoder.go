// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	hcl "github.com/hashicorp/hcl/v2"
)

type matrixDecoder struct{}

type DecodedMatrices struct {
	Original        *Matrix
	IncludeProducts []*Matrix
	Excludes        []*Exclude
	FinalProduct    *Matrix
}

func newMatrixDecoder() *matrixDecoder {
	return &matrixDecoder{}
}

func (d *DecodedMatrices) Matrix() *Matrix {
	if d == nil {
		return nil
	}

	return d.FinalProduct
}

func (d *DecodedMatrices) Filter(f *ScenarioFilter) *Matrix {
	if d == nil || d.FinalProduct == nil {
		return nil
	}

	return d.FinalProduct.Filter(f)
}

func (d *DecodedMatrices) Set(m *Matrix) {
	if d == nil {
		return
	}

	d.FinalProduct = m
}

func (md *matrixDecoder) decodeMatrixAttribute(
	ctx *hcl.EvalContext,
	block *hcl.Block,
	attr *hcl.Attribute,
) (*Vector, hcl.Diagnostics) {
	diags := hcl.Diagnostics{}
	vec := NewVector()

	val, moreDiags := attr.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return vec, diags
	}

	if !val.CanIterateElements() {
		return vec, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "matrix attribute value must be a list of strings",
			Detail:   fmt.Sprintf("expected value for %s to be a list of strings, found %s", attr.Name, val.Type().GoString()),
			Subject:  attr.NameRange.Ptr(),
			Context:  block.DefRange.Ptr(),
		})
	}

	if len(val.AsValueSlice()) == 0 {
		return vec, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "matrix attribute values cannot be empty lists",
			Subject:  attr.NameRange.Ptr(),
			Context:  block.DefRange.Ptr(),
		})
	}

	for _, elm := range val.AsValueSlice() {
		if !elm.Type().Equals(cty.String) {
			return vec, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "matrix attribute value must be a list of strings",
				Detail:   "found element with type " + elm.GoString(),
				Subject:  attr.NameRange.Ptr(),
				Context:  block.DefRange.Ptr(),
			})
		}

		vec.Add(NewElement(attr.Name, elm.AsString()))
	}

	return vec, diags
}

// Go maps are intentionally unordered. We need to sort our attributes
// so that our variants elements are deterministic every time we
// decode our flightplan.
func (md *matrixDecoder) sortAttributes(attrs map[string]*hcl.Attribute) []*hcl.Attribute {
	sorted := []*hcl.Attribute{}
	for _, attr := range attrs {
		sorted = append(sorted, attr)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

// decodeMatrix takes an eval context and scenario blocks and decodes only the matrix block.
// As the matrix block can contain variants, includes, and excludes, the response will contain
// the original variants as a Matrix with those vectors, the decoded includes, decoded excludes
// and the final cartesian product of unique values.
func (md *matrixDecoder) decodeMatrix(
	ctx *hcl.EvalContext,
	block *hcl.Block,
) (*DecodedMatrices, hcl.Diagnostics) {
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
	matrix := NewMatrix()
	res := &DecodedMatrices{Original: matrix}

	// Each attribute in the matrix should be a variant name whose value must
	// be a list of strings. Convert the value into a matrix vector and add it.
	// We're ignoring the diagnostics JustAttributes() will return here because
	// there might also be include and exclude blocks.
	mAttrs, _ := block.Body.JustAttributes()
	for _, attr := range md.sortAttributes(mAttrs) {
		vec, moreDiags := md.decodeMatrixAttribute(ctx, block, attr)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		res.Original.AddVector(vec)
	}

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
			iMatrix := NewMatrix()
			iAttrs, moreDiags := mBlock.Body.JustAttributes()
			diags = diags.Extend(moreDiags)
			if moreDiags != nil && moreDiags.HasErrors() {
				continue
			}

			for _, attr := range md.sortAttributes(iAttrs) {
				vec, moreDiags := md.decodeMatrixAttribute(ctx, mBlock, attr)
				diags = diags.Extend(moreDiags)
				if moreDiags != nil && moreDiags.HasErrors() {
					continue
				}

				iMatrix.AddVector(vec)
			}

			// Generate our possible include vectors and add them to our main matrix.
			iMatrixProduct := iMatrix.CartesianProduct().UniqueValues()
			res.IncludeProducts = append(res.IncludeProducts, iMatrixProduct)

			for _, vec := range iMatrixProduct.GetVectors() {
				res.FinalProduct.AddVector(vec)
			}
		case "exclude":
			eMatrix := NewMatrix()
			eAttrs, moreDiags := mBlock.Body.JustAttributes()
			diags = diags.Extend(moreDiags)
			if moreDiags != nil && moreDiags.HasErrors() {
				continue
			}

			for _, attr := range md.sortAttributes(eAttrs) {
				vec, moreDiags := md.decodeMatrixAttribute(ctx, mBlock, attr)
				diags = diags.Extend(moreDiags)
				if moreDiags != nil && moreDiags.HasErrors() {
					continue
				}
				eMatrix.AddVector(vec)
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

	return res, diags
}