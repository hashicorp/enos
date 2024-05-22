// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"cmp"
	"errors"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	hcl "github.com/hashicorp/hcl/v2"
)

var qualitySchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "description", Required: true},
	},
}

// Quality represents an Enos Terraform quality block.
type Quality struct {
	Name        string
	Description string
}

// NewQuality returns a new Quality.
func NewQuality() *Quality {
	return &Quality{}
}

// CompareQuality implements a slices.SortFunc for Quality's.
func CompareQuality(a, b *Quality) int {
	if a == nil && b == nil {
		return 0
	}

	if a != nil && b == nil {
		return 1
	}

	if a == nil && b != nil {
		return -1
	}

	if n := cmp.Compare(a.Name, b.Name); n != 0 {
		return n
	}

	return cmp.Compare(a.Description, b.Description)
}

// CompareQuality implements a slices.SortFunc for pb.Quality's.
func CompareQualityProto(ap, bp *pb.Quality) int {
	a := NewQuality()
	b := NewQuality()
	a.FromProto(ap)
	b.FromProto(bp)

	return CompareQuality(a, b)
}

// ToProto coverts the Quality struct to the wire representation.
func (q *Quality) ToProto() *pb.Quality {
	if q == nil {
		return nil
	}

	return &pb.Quality{
		Name:        q.Name,
		Description: q.Description,
	}
}

// FromProto unmarshal the Quality wire representation onto the struct.
func (q *Quality) FromProto(p *pb.Quality) {
	if q == nil || p == nil {
		return
	}

	q.Name = p.GetName()
	q.Description = p.GetDescription()
}

// ToCtyValue returns the quality contents as an object cty.Value. We can then
// embed this into the quality section of the eval context to allowed method
// style expression references.
func (q *Quality) ToCtyValue() cty.Value {
	vals := map[string]cty.Value{
		"name":        cty.StringVal(q.Name),
		"description": cty.StringVal(q.Description),
	}

	return cty.ObjectVal(vals)
}

// FromCtyValue takes a cty.Value and unmarshals it onto itself. It expects
// a valid object created from ToCtyValue().
func (q *Quality) FromCtyValue(val cty.Value) error {
	if val.IsNull() {
		return nil
	}

	if !val.IsWhollyKnown() {
		return errors.New("cannot unmarshal unknown value")
	}

	if !val.Type().IsObjectType() {
		return errors.New("value must be an object")
	}

	for key, val := range val.AsValueMap() {
		switch key {
		case "description":
			if val.Type() != cty.String {
				return errors.New("source must be a string")
			}
			q.Description = val.AsString()
		case "name":
			if val.Type() != cty.String {
				return errors.New("name must be a string")
			}
			q.Name = val.AsString()
		default:
			return fmt.Errorf("unknown attribute '%s", key)
		}
	}

	return nil
}

// decode takes in an HCL block of a quality and an eval context and decodes from
// the block onto itself. Any errors that are encountered are returned as hcl
// diagnostics.
func (q *Quality) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	content, moreDiags := block.Body.Content(qualitySchema)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	q.Name = block.Labels[0]

	for name, attr := range content.Attributes {
		switch name {
		case "description":
			val, moreDiags := attr.Expr.Value(ctx)
			diags = diags.Extend(moreDiags)
			if moreDiags != nil && moreDiags.HasErrors() {
				continue
			}
			q.Description = val.AsString()
		default:
			// This should never happen since our content should return an error if unsupported attrs
			// are present.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unknown attribute " + name + " in 'quality' block",
				Subject:  attr.NameRange.Ptr(),
			})
		}
	}

	return diags
}
