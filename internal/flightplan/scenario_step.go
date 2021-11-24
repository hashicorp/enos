package flightplan

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
)

// scenarioStepSchema is our knowable scenario step schema.
var scenarioStepSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "module", Required: true},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "variables"},
	},
}

// ScenarioStep is a step in an Enos scenario
type ScenarioStep struct {
	Name   string
	Module *ScenarioStepModule
}

// ScenarioStepModule is the module attribute value in a scenario step
type ScenarioStepModule struct {
	Name    string
	Source  string
	Version string
	Attrs   map[string]cty.Value
}

// NewScenarioStep returns a new Scenario step
func NewScenarioStep() *ScenarioStep {
	return &ScenarioStep{
		Module: &ScenarioStepModule{
			Attrs: map[string]cty.Value{},
		},
	}
}

// decode takes an HCL block and eval context and decodes itself from the block.
// It's responsible for ensuring that the block contains a "module" attribute
// and that it references an existing module that has been previously defined.
// It performs module reference validation by comparing our module reference
// to defined modules that are available in the eval context variable "module".
// We then inherit the default variables from the module reference and then
// evaluate our own "variables" block to get step level attributes.
func (ss *ScenarioStep) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// Decode the our scenario step
	content, moreDiags := block.Body.Content(scenarioStepSchema)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	// Decode our name
	ss.Name = block.Labels[0]

	// Decode the step module reference
	moduleAttr, moreDiags := ss.decodeModuleAttribute(block, content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	// Validate that our module references an existing module in the eval context.
	moduleVal, moreDiags := ss.validateModuleAttributeReference(moduleAttr, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	// Copy variable attributes from the module to our step. This is how we'll
	// inherit module variables and their values.
	ss.copyModuleAttributes(moduleVal)

	// Decode step variables. This will decode all variables and set them or
	// override any inherited values from the module.
	diags = diags.Extend(ss.decodeVariables(content.Blocks.OfType("variables"), ctx))

	return diags
}

// decodeModuleAttribute decodes the module attribute from the content and ensures
// that it has the required source and name fields. It returns the HCL attribute
// for further validation later.
func (ss *ScenarioStep) decodeModuleAttribute(block *hcl.Block, content *hcl.BodyContent, ctx *hcl.EvalContext) (*hcl.Attribute, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	module, ok := content.Attributes["module"]
	if !ok {
		r := block.Body.MissingItemRange()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "scenario step missing module",
			Detail:   "scenario step missing module",
			Subject:  &r,
		})

		return module, diags
	}

	// Now that we've found our module attribute we need to decode it and
	// validate that it's referring to an allowed module, i.e. one that has
	// been defined at the top-level and has a matching name and source.
	val, moreDiags := module.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return module, diags
	}

	// Since we don't know the variable schema of the Terraform module we're
	// referencing we have to manually decode the parts that we do know. Everything
	// else we'll pass along to Terraform.
	if val.IsNull() || !val.IsWhollyKnown() || !val.CanIterateElements() {
		r := module.Expr.Range()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid module value",
			Detail:   "module must be a known module object",
			Subject:  &r,
		})

		return module, diags
	}

	name, ok := val.AsValueMap()["name"]
	if !ok {
		r := module.Expr.Range()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "missing module name",
			Detail:   "missing module name",
			Subject:  &r,
		})

		return module, diags
	}
	ss.Module.Name = name.AsString()

	source, ok := val.AsValueMap()["source"]
	if !ok {
		r := module.Expr.Range()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "missing module source",
			Detail:   "missing module source",
			Subject:  &r,
		})

		return module, diags
	}
	ss.Module.Source = source.AsString()

	// version is not required so we'll only get it if it has been defined.
	version, ok := val.AsValueMap()["version"]
	if ok {
		ss.Module.Version = version.AsString()
	}

	return module, diags
}

func (ss *ScenarioStep) validateModuleAttributeReference(module *hcl.Attribute, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var modules cty.Value
	var foundModules bool
	var moduleVal cty.Value

	// Search through the eval context chain until we find a "modules" variable.
	for modCtx := ctx; modCtx != nil; modCtx = modCtx.Parent() {
		if modCtx == nil {
			r := module.Expr.Range()
			diags = diags.Append(&hcl.Diagnostic{

				Severity: hcl.DiagError,
				Summary:  "unknown module",
				Detail:   fmt.Sprintf("a module with name %s has not been defined", ss.Module.Name),
				Subject:  &r,
			})

			return moduleVal, diags
		}

		m, ok := modCtx.Variables["module"]
		if ok {
			modules = m
			foundModules = true
			break
		}
	}

	if !foundModules {
		r := module.Expr.Range()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unknown module",
			Detail:   "cannot use module as no modules have been found declared",
			Subject:  &r,
		})

		return moduleVal, diags
	}

	// Validate that our module configuration references an existing module.
	// We only care that the name and source match. All other variables and
	// and attributes we'll carry over.
	for _, mod := range modules.AsValueSlice() {
		name, ok := mod.AsValueMap()["name"]
		if !ok {
			// This should never happen
			continue
		}

		if name.AsString() != ss.Module.Name {
			continue
		}

		source, ok := mod.AsValueMap()["source"]
		if !ok {
			r := module.Expr.Range()
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "missing source",
				Detail:   "module value does not contain a source",
				Subject:  &r,
			})

			break
		}

		if s := source.AsString(); s != ss.Module.Source {
			r := module.Expr.Range()
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "module source doesn't match module definition",
				Detail: fmt.Sprintf("module source for module %s is %s, not %s",
					ss.Module.Name, s, ss.Module.Source,
				),
				Subject: &r,
			})
			break
		}

		moduleVal = mod
		break
	}

	if moduleVal.IsNull() {
		r := module.Expr.Range()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unknown module",
			Detail:   fmt.Sprintf("a module with name %s has not been defined", ss.Module.Name),
			Subject:  &r,
		})

		return moduleVal, diags
	}

	return moduleVal, diags
}

func (ss *ScenarioStep) copyModuleAttributes(module cty.Value) {
	isReservedAttr := func(name string) bool {
		for _, reserved := range []string{"name", "source", "version"} {
			if name == reserved {
				return true
			}
		}
		return false
	}

	for name, value := range module.AsValueMap() {
		if isReservedAttr(name) {
			continue
		}

		ss.Module.Attrs[name] = value
	}
}

func (ss *ScenarioStep) decodeVariables(varBlocks hcl.Blocks, ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	for _, varBlock := range varBlocks {
		attrs, moreDiags := varBlock.Body.JustAttributes()
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
			ss.Module.Attrs[attr.Name], moreDiags = attr.Expr.Value(ctx)
			diags = diags.Extend(moreDiags)
		}

		diags = diags.Extend(verifyNoBlockInAttrOnlySchema(varBlock.Body))
	}

	return diags
}
