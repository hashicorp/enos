package flightplan

import (
	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
)

var moduleSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "source", Required: true},
		{Name: "version"},
	},
}

// Module represents an Enos Terraform module block
type Module struct {
	Name    string
	Source  string
	Version string
	Attrs   map[string]cty.Value
}

// NewModule returns a new Module
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
	val, moreDiags := content.Attributes["source"].Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}
	m.Source = val.AsString()

	// "version" isn't required attribute but it is allowed. Handle it manually.
	version, ok := content.Attributes["version"]
	if ok {
		val, moreDiags := version.Expr.Value(ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			return diags
		}
		m.Version = val.AsString()
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

// evalCtx returns the module contents as an object cty.Value. We can then
// embed this into the Variables section of the eval context to allowed method
// style expression references.
func (m *Module) evalCtx() cty.Value {
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
