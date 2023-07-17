package flightplan

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	hcl "github.com/hashicorp/hcl/v2"
)

var moduleSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "source", Required: true},
		{Name: "version"},
	},
}

// Module represents an Enos Terraform module block.
type Module struct {
	Name    string
	Source  string
	Version string
	Attrs   map[string]cty.Value
}

// NewModule returns a new Module.
func NewModule() *Module {
	return &Module{
		Attrs: map[string]cty.Value{},
	}
}

// decode takes in an HCL block of a module and an eval context and decodes from
// the block onto itself. Any errors that are encountered are returned as hcl
// diagnostics.
func (m *Module) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	content, remain, moreDiags := block.Body.PartialContent(moduleSchema)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	m.Name = block.Labels[0]
	src := content.Attributes["source"]
	val, moreDiags := src.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	if val.Type() == cty.String {
		m.Source = val.AsString()
	} else {
		sourceVal, err := convert.Convert(val, cty.String)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "invalid value",
				Detail:   "source must be a string value",
				Subject:  src.Expr.Range().Ptr(),
				Context:  hcl.RangeBetween(src.Expr.StartRange(), src.Expr.Range()).Ptr(),
			})
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "invalid value",
				Detail:   "source should be a string value, consider changing it",
				Subject:  src.Expr.Range().Ptr(),
				Context:  hcl.RangeBetween(src.Expr.StartRange(), src.Expr.Range()).Ptr(),
			})
			m.Source = sourceVal.AsString()
		}
	}

	// "version" isn't required attribute but it is allowed. Handle it manually.
	version, ok := content.Attributes["version"]
	if ok {
		val, moreDiags := version.Expr.Value(ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			return diags
		}

		if val.Type() == cty.String {
			m.Version = val.AsString()
		} else {
			versionVal, err := convert.Convert(val, cty.String)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "invalid value",
					Detail:   "version must be a string value",
					Subject:  version.Expr.Range().Ptr(),
					Context:  hcl.RangeBetween(version.Expr.StartRange(), version.Expr.Range()).Ptr(),
				})
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "invalid value",
					Detail:   "version should be a string value, consider changing it",
					Subject:  version.Expr.Range().Ptr(),
					Context:  hcl.RangeBetween(version.Expr.StartRange(), version.Expr.Range()).Ptr(),
				})
				m.Version = versionVal.AsString()
			}
		}
	}

	// The remaining portion of our module block is a bunch of attributes
	// that we'll pass to Terraform in our generated module. We don't know the
	// schema, nor do we need to, we only need to know the cty.Value of each
	// attribute.
	attrs, moreDiags := remain.JustAttributes()
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	attrs, moreDiags = filterTerraformMetaAttrs(attrs)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	for _, attr := range attrs {
		m.Attrs[attr.Name], moreDiags = attr.Expr.Value(ctx)
		diags = diags.Extend(moreDiags)
	}

	diags = diags.Extend(verifyNoBlockInAttrOnlySchema(remain))

	return diags
}

// ToCtyValue returns the module contents as an object cty.Value. We can then
// embed this into the Variables section of the eval context to allowed method
// style expression references.
func (m *Module) ToCtyValue() cty.Value {
	vals := map[string]cty.Value{
		"source": cty.StringVal(m.Source),
		"name":   cty.StringVal(m.Name),
	}
	if m.Version != "" {
		vals["version"] = cty.StringVal(m.Version)
	}

	for k, v := range m.Attrs {
		vals[k] = v
	}

	return cty.ObjectVal(vals)
}

// FromCtyValue takes a cty.Value and unmarshals it onto itself. It expects
// a valid object created from ToCtyValue().
func (m *Module) FromCtyValue(val cty.Value) error {
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
		case "source":
			if val.Type() != cty.String {
				return fmt.Errorf("source must be a string")
			}
			m.Source = val.AsString()
		case "name":
			if val.Type() != cty.String {
				return fmt.Errorf("name must be a string ")
			}
			m.Name = val.AsString()
		case "version":
			if val.Type() != cty.String {
				return fmt.Errorf("version must be a string ")
			}
			m.Version = val.AsString()
		default:
			m.Attrs[key] = val
		}
	}

	return nil
}
