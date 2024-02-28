package flightplan

import (
	"errors"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
)

// Provider is a Enos transport configuration.
type Provider struct {
	Type   string           `cty:"type"`
	Alias  string           `cty:"alias"`
	Config *SchemalessBlock `cty:"config"`
}

// NewProvider returns a new Provider.
func NewProvider() *Provider {
	return &Provider{
		Config: NewSchemalessBlock(),
	}
}

// decode takes in an HCL block of a provider and an eval context and
// decodes from the block onto itself. Any errors that are encountered are
// returned as hcl diagnostics.
func (p *Provider) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	p.Type = block.Labels[0]
	p.Alias = block.Labels[1]

	// Decode the entire provider block as a schemaless block
	moreDiags := p.Config.Decode(block, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	return diags
}

// ToCtyValue returns the provider contents as an object cty.Value.
func (p *Provider) ToCtyValue() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"type":   cty.StringVal(p.Type),
		"alias":  cty.StringVal(p.Alias),
		"config": p.Config.ToCtyValue(),
	})
}

// FromCtyValue takes a cty.Value and unmarshals it onto itself. It expects
// a valid object created from ToCtyValue().
func (p *Provider) FromCtyValue(val cty.Value) error {
	var err error

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
				return errors.New("provider type must be a string")
			}
			p.Type = val.AsString()
		case "alias":
			if val.Type() != cty.String {
				return errors.New("provider alias must be a string ")
			}
			p.Alias = val.AsString()
		case "config":
			err = p.Config.FromCtyValue(val)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown key in value object: %s", key)
		}
	}

	return nil
}
