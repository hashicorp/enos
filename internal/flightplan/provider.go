package flightplan

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
)

// Provider is a Enos transport configuration
type Provider struct {
	Type  string               `cty:"type"`
	Alias string               `cty:"alias"`
	Attrs map[string]cty.Value `cty:"attrs"`
}

// NewProvider returns a new Provider
func NewProvider() *Provider {
	return &Provider{
		Attrs: map[string]cty.Value{},
	}
}

// decode takes in an HCL block of a provider and an eval context and
// decodes from the block onto itself. Any errors that are encountered are
// returned as hcl diagnostics.
func (p *Provider) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	p.Type = block.Labels[0]
	p.Alias = block.Labels[1]

	if p.Type == "enos" {
		// Since we know the schema for the "enos" provider we can more fine
		// grained decoding.
		moreDiags := p.decodeEnosProvider(block, ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			return diags
		}
	} else {
		attrs, moreDiags := block.Body.JustAttributes()
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			return diags
		}

		for name, attr := range attrs {
			val, moreDiags := attr.Expr.Value(ctx)
			diags = diags.Extend(moreDiags)
			if moreDiags.HasErrors() {
				continue
			}
			p.Attrs[name] = val
		}
	}

	return diags
}

// ToCtyValue returns the provider contents as an object cty.Value.
func (p *Provider) ToCtyValue() cty.Value {
	vals := map[string]cty.Value{
		"type":  cty.StringVal(p.Type),
		"alias": cty.StringVal(p.Alias),
	}

	if len(p.Attrs) > 0 {
		vals["attrs"] = cty.ObjectVal(p.Attrs)
	} else {
		vals["attrs"] = cty.NullVal(cty.EmptyObject)
	}

	return cty.ObjectVal(vals)
}

// FromCtyValue takes a cty.Value and unmarshals it onto itself. It expects
// a valid object created from ToCtyValue()
func (p *Provider) FromCtyValue(val cty.Value) error {
	if val.IsNull() {
		return nil
	}

	if !val.IsWhollyKnown() {
		return fmt.Errorf("cannot unmarshal unknown value")
	}

	if !val.CanIterateElements() {
		return fmt.Errorf("value must be an object")
	}

	for key, val := range val.AsValueMap() {
		switch key {
		case "type":
			if val.Type() != cty.String {
				return fmt.Errorf("provider type must be a string")
			}
			p.Type = val.AsString()
		case "alias":
			if val.Type() != cty.String {
				return fmt.Errorf("provider alias must be a string ")
			}
			p.Alias = val.AsString()
		case "attrs":
			if !val.CanIterateElements() {
				return fmt.Errorf("provider attrs must a map of attributes")
			}

			for k, v := range val.AsValueMap() {
				p.Attrs[k] = v
			}
		default:
			return fmt.Errorf("unknown key in value object: %s", key)
		}
	}

	return nil
}

func (p *Provider) decodeEnosProvider(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// Validate that our configuration matches our allowed schema.
	spec := hcldec.ObjectSpec{
		"transport": &hcldec.AttrSpec{
			Name:     "transport",
			Required: false,
			Type: cty.ObjectWithOptionalAttrs(map[string]cty.Type{
				"ssh": cty.ObjectWithOptionalAttrs(map[string]cty.Type{
					"user":             cty.String,
					"host":             cty.String,
					"private_key":      cty.String,
					"private_key_path": cty.String,
					"passphrase":       cty.String,
					"passphrase_path":  cty.String,
				}, []string{
					"user", "host", "private_key", "private_key_path",
					"passphrase", "passphrase_path",
				}),
			}, []string{"ssh"}),
		},
	}

	val, moreDiags := hcldec.Decode(block.Body, spec, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	if val.IsNull() || !val.IsWhollyKnown() || !val.CanIterateElements() {
		return diags
	}

	trans, ok := val.AsValueMap()["transport"]
	if !ok {
		return diags
	}

	if trans.IsNull() || !trans.IsWhollyKnown() || !trans.CanIterateElements() {
		return diags
	}

	ssh, ok := trans.AsValueMap()["ssh"]
	if !ok {
		return diags
	}

	if ssh.IsNull() || !ssh.IsWhollyKnown() || !ssh.CanIterateElements() {
		return diags
	}

	sshVals := map[string]cty.Value{}
	for name, val := range ssh.AsValueMap() {
		// Only pass through known values
		if val.IsNull() || !val.IsWhollyKnown() {
			continue
		}

		switch name {
		case "passphrase_path", "private_key_path":
			// Since these are set and they're paths we can ensure that that
			// they actually exist.
			abs, err := filepath.Abs(val.AsString())
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("expanding path for enos provider %s", name),
					Detail:   err.Error(),
					Subject:  block.TypeRange.Ptr(),
					Context:  block.DefRange.Ptr(),
				})
				continue
			}

			bytes, err := os.ReadFile(abs)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("reading the file contents for enos provider %s", name),
					Detail:   err.Error(),
					Subject:  block.TypeRange.Ptr(),
					Context:  block.DefRange.Ptr(),
				})
				continue
			}

			if len(bytes) == 0 {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("empty content not allowed for enos provider %s", name),
					Detail:   err.Error(),
					Subject:  block.TypeRange.Ptr(),
					Context:  block.DefRange.Ptr(),
				})
				continue
			}
			sshVals[name] = cty.StringVal(abs)
		default:
			sshVals[name] = val
		}
	}

	p.Attrs["transport"] = cty.ObjectVal(map[string]cty.Value{
		"ssh": cty.ObjectVal(sshVals),
	})

	return diags
}
