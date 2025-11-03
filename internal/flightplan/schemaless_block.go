// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"errors"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// SchemalessBlock is our value on HCL block that has no known schema.
type SchemalessBlock struct {
	Type     string               `cty:"type"`
	Labels   []string             `cty:"labels"`
	Attrs    map[string]cty.Value `cty:"attrs"`
	Children []*SchemalessBlock   `cty:"blocks"`
}

// NewSchemalessBlock takes a block type and any labels and returns a new
// schemaless block.
func NewSchemalessBlock() *SchemalessBlock {
	return &SchemalessBlock{
		Labels:   []string{},
		Attrs:    map[string]cty.Value{},
		Children: []*SchemalessBlock{},
	}
}

// Decode takes in an HCL block and eval context and attempts to decode and
// evaluate it.
func (s *SchemalessBlock) Decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	if block == nil {
		return diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "cannot decode nil block",
			EvalContext: ctx,
		})
	}

	s.Type = block.Type
	s.Labels = block.Labels

	// We need to cast this to an hclsyntax body to get access to the blocks.
	// It also helps us get the attribute values without generating diagnostics.
	body, ok := block.Body.(*hclsyntax.Body)
	if !ok {
		// This should never happen
		return diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "unable to decode block",
			Detail:      "unable to cast block to the hclsyntax",
			Subject:     block.TypeRange.Ptr(),
			EvalContext: ctx,
		})
	}

	for name, attr := range body.Attributes {
		val, moreDiags := attr.AsHCLAttribute().Expr.Value(ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}
		s.Attrs[name] = val
	}

	for _, child := range body.Blocks {
		csb := NewSchemalessBlock()
		moreDiags := csb.Decode(child.AsHCLBlock(), ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			return diags
		}
		s.Children = append(s.Children, csb)
	}

	return diags
}

// ToCtyValue returns the schemaless block contents as an object cty.Value.
func (s *SchemalessBlock) ToCtyValue() cty.Value {
	vals := map[string]cty.Value{
		"type":   cty.StringVal(s.Type),
		"labels": cty.ListValEmpty(cty.String),
		"attrs":  cty.NullVal(cty.EmptyObject),
		"blocks": cty.ListValEmpty(cty.EmptyObject),
	}

	labels := []cty.Value{}
	for _, l := range s.Labels {
		labels = append(labels, cty.StringVal(l))
	}
	if len(labels) > 0 {
		vals["labels"] = cty.ListVal(labels)
	}

	if len(s.Attrs) > 0 {
		vals["attrs"] = cty.ObjectVal(s.Attrs)
	}

	if len(s.Children) > 0 {
		blocks := []cty.Value{}
		for _, b := range s.Children {
			blocks = append(blocks, b.ToCtyValue())
		}
		vals["blocks"] = cty.ListVal(blocks)
	}

	return cty.ObjectVal(vals)
}

// FromCtyValue takes a cty.Value and unmarshals it onto itself. It expects
// a valid object created from ToCtyValue().
func (s *SchemalessBlock) FromCtyValue(val cty.Value) error {
	if val.IsNull() {
		return nil
	}

	if !val.IsWhollyKnown() {
		return errors.New("cannot unmarshal unknown value")
	}

	if !val.CanIterateElements() {
		return errors.New("value must be an object")
	}

	for key, val := range val.AsValueMap() {
		switch key {
		case "type":
			if val.Type() != cty.String {
				return errors.New("block type must be a string")
			}
			s.Type = val.AsString()
		case "labels":
			if val.Type() != cty.List(cty.String) {
				return errors.New("block aliases must be a list of strings")
			}
			s.Labels = []string{}
			for _, v := range val.AsValueSlice() {
				if !v.IsWhollyKnown() || v.IsNull() {
					continue
				}
				s.Labels = append(s.Labels, v.AsString())
			}
		case "attrs":
			if !val.CanIterateElements() {
				return errors.New("provider attrs must a map of attributes")
			}

			for k, v := range val.AsValueMap() {
				s.Attrs[k] = v
			}
		case "blocks":
			if !val.CanIterateElements() {
				return errors.New("provider blocks must be a list of blocks")
			}

			for _, v := range val.AsValueSlice() {
				sb := NewSchemalessBlock()
				err := sb.FromCtyValue(v)
				if err != nil {
					return err
				}
				s.Children = append(s.Children, sb)
			}
		default:
			return fmt.Errorf("unknown key in value object: %s", key)
		}
	}

	return nil
}
