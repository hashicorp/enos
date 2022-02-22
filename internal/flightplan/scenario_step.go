package flightplan

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	hcl "github.com/hashicorp/hcl/v2"
)

// scenarioStepSchema is our knowable scenario step schema.
var scenarioStepSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "module", Required: true},
		{Name: "providers", Required: false},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "variables"},
	},
}

// ScenarioStep is a step in an Enos scenario
type ScenarioStep struct {
	Name      string
	Module    *Module
	Providers map[string]*Provider
}

// NewScenarioStep returns a new Scenario step
func NewScenarioStep() *ScenarioStep {
	return &ScenarioStep{
		Module: NewModule(),
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

	// Handle our named providers, if any
	moreDiags = ss.decodeAndValidateProvidersAttribute(content, ctx)
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
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "scenario step missing module",
			Detail:   "scenario step missing module",
			Subject:  block.Body.MissingItemRange().Ptr(),
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
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid module value",
			Detail:   "module must be a known module object",
			Subject:  module.Expr.Range().Ptr(),
			Context:  hcl.RangeBetween(module.NameRange, module.Expr.Range()).Ptr(),
		})

		return module, diags
	}

	name, ok := val.AsValueMap()["name"]
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "missing module name",
			Detail:   "missing module name",
			Subject:  module.NameRange.Ptr(),
			Context:  hcl.RangeBetween(module.NameRange, module.Expr.Range()).Ptr(),
		})

		return module, diags
	}
	ss.Module.Name = name.AsString()

	source, ok := val.AsValueMap()["source"]
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "missing module source",
			Detail:   "missing module source",
			Subject:  module.Expr.Range().Ptr(),
			Context:  hcl.RangeBetween(module.Range, module.Expr.Range()).Ptr(),
		})

		return module, diags
	}
	if source.Type() == cty.String {
		ss.Module.Source = source.AsString()
	} else {
		sourceVal, err := convert.Convert(source, cty.String)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "invalid module source value",
				Detail:   "module source value must be a string",
				Subject:  module.Expr.Range().Ptr(),
				Context:  hcl.RangeBetween(module.Range, module.Expr.Range()).Ptr(),
			})
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "invalid module source value",
				Detail:   "module source value should be a string, consider updating it",
				Subject:  module.Expr.Range().Ptr(),
				Context:  hcl.RangeBetween(module.Range, module.Expr.Range()).Ptr(),
			})
			ss.Module.Source = sourceVal.AsString()
		}
	}

	// version is not required so we'll only get it if it has been defined.
	version, ok := val.AsValueMap()["version"]
	if ok {
		if version.Type() == cty.String {
			ss.Module.Version = version.AsString()
		} else {
			versionVal, err := convert.Convert(version, cty.String)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "invalid module version value",
					Detail:   "module version value must be a string",
					Subject:  module.Expr.Range().Ptr(),
					Context:  hcl.RangeBetween(module.Range, module.Expr.Range()).Ptr(),
				})
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "invalid module version value",
					Detail:   "module version value should be a string, consider updating it",
					Subject:  module.Expr.Range().Ptr(),
					Context:  hcl.RangeBetween(module.Range, module.Expr.Range()).Ptr(),
				})
				ss.Module.Version = versionVal.AsString()
			}
		}
	}

	return module, diags
}

func (ss *ScenarioStep) validateModuleAttributeReference(module *hcl.Attribute, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var modules cty.Value
	var moduleVal cty.Value
	var err error

	// Search through the eval context chain until we find a "modules" variable.
	modules, err = findEvalContextVariable("module", ctx)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unknown module",
			Detail:   fmt.Sprintf("a module with name %s has not been defined", ss.Module.Name),
			Subject:  module.Expr.Range().Ptr(),
			Context:  hcl.RangeBetween(module.NameRange, module.Expr.Range()).Ptr(),
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
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "missing source",
				Detail:   "module value does not contain a source",
				Subject:  module.Expr.Range().Ptr(),
				Context:  hcl.RangeBetween(module.NameRange, module.Expr.Range()).Ptr(),
			})

			break
		}

		if s := source.AsString(); s != ss.Module.Source {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "module source doesn't match module definition",
				Detail: fmt.Sprintf("module source for module %s is %s, not %s",
					ss.Module.Name, s, ss.Module.Source,
				),
				Subject: module.Expr.Range().Ptr(),
				Context: hcl.RangeBetween(module.NameRange, module.Expr.Range()).Ptr(),
			})
			break
		}

		moduleVal = mod
		break
	}

	if moduleVal.IsNull() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unknown module",
			Detail:   fmt.Sprintf("a module with name %s has not been defined", ss.Module.Name),
			Subject:  module.Expr.Range().Ptr(),
			Context:  hcl.RangeBetween(module.NameRange, module.Expr.Range()).Ptr(),
		})

		return moduleVal, diags
	}

	return moduleVal, diags
}

// decodeAndValidateProvidersAttribute decodess the providers attribute
// from the content and validates that each sub-attribute references a defined
// provider.
func (ss *ScenarioStep) decodeAndValidateProvidersAttribute(content *hcl.BodyContent, ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	providers, ok := content.Attributes["providers"]
	if !ok {
		return diags
	}

	ss.Providers = map[string]*Provider{}

	providersVal, moreDiags := providers.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	if providersVal.IsNull() || !providersVal.IsWhollyKnown() || !providersVal.CanIterateElements() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "providers value must be a known object",
			Subject:  providers.Expr.Range().Ptr(),
			Context:  providers.Range.Ptr(),
		})
	}

	// Get our defined providers from the eval context
	definedProviders, err := findEvalContextVariable("provider", ctx)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "provider value has not been defined",
			Detail:   err.Error(),
			Subject:  providers.Expr.Range().Ptr(),
			Context:  providers.Range.Ptr(),
		})
	}

	// Unroll them so we can look up our provider values by type and alias
	unrolled := map[string]map[string]cty.Value{}
	if definedProviders.IsNull() || !definedProviders.IsWhollyKnown() || !definedProviders.CanIterateElements() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "cannot set provider as no providers have been defined",
			Subject:  providers.Expr.Range().Ptr(),
			Context:  providers.Range.Ptr(),
		})
	}
	for pType, pVals := range definedProviders.AsValueMap() {
		aliases := map[string]cty.Value{}
		for alias, aliasV := range pVals.AsValueMap() {
			aliases[alias] = aliasV
		}
		unrolled[pType] = aliases
	}

	// For each defined provider, make sure a matching instance is defined and
	// matches
	for providerImportName, providerVal := range providersVal.AsValueMap() {
		provider := NewProvider()
		err := provider.FromCtyValue(providerVal)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("unable to unmarshal provider value for %s", providerImportName),
				Detail:   err.Error(),
				Subject:  providers.Expr.Range().Ptr(),
				Context:  providers.Range.Ptr(),
			})
			continue
		}

		types, ok := unrolled[provider.Type]
		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("provider type %s is not defined", provider.Type),
				Subject:  providers.Expr.Range().Ptr(),
				Context:  providers.Range.Ptr(),
			})
			continue
		}

		alias, ok := types[provider.Alias]
		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("alias %s for provider type %s is not defined", provider.Alias, provider.Type),
				Subject:  providers.Expr.Range().Ptr(),
				Context:  providers.Range.Ptr(),
			})
			continue
		}

		if providerVal.Equals(alias) != cty.True {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "provider arguments don't match defined provider",
				Subject:  providers.Expr.Range().Ptr(),
				Context:  providers.Range.Ptr(),
			})
		}

		ss.Providers[providerImportName] = provider
	}

	return diags
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
