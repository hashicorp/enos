// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
)

// ScenarioOutput represents an "output" block in a scenario.
type ScenarioOutput struct {
	Name        string
	Description string
	Sensitive   bool
	Value       cty.Value
}

// NewScenarioOutput returns a new Output.
func NewScenarioOutput() *ScenarioOutput {
	return &ScenarioOutput{Value: cty.NilVal}
}

// decode takes in an HCL block of an output and unmarshal the value into itself.
func (v *ScenarioOutput) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}
	v.Name = block.Labels[0]

	// Define this here so StepVariableType is our cty.Type and not cty.Nil
	scenarioOutputSpec := hcldec.ObjectSpec{
		"description": &hcldec.AttrSpec{
			Name:     "description",
			Type:     cty.String,
			Required: false,
		},
		"sensitive": &hcldec.AttrSpec{
			Name:     "sensitive",
			Type:     cty.Bool,
			Required: false,
		},
		"value": &hcldec.AttrSpec{
			Name:     "value",
			Type:     StepVariableType,
			Required: true,
		},
	}

	val, moreDiags := hcldec.Decode(block.Body, scenarioOutputSpec, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	for attr, val := range val.AsValueMap() {
		if val.IsNull() || !val.IsWhollyKnown() {
			continue
		}

		switch attr {
		case "description":
			v.Description = val.AsString()
		case "sensitive":
			v.Sensitive = val.True()
		case "value":
			v.Value = val
		default:
		}
	}

	return diags
}
