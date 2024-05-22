// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
)

// scenarioStepSchema is our knowable scenario step schema.
var scenarioStepSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "description", Required: false},
		{Name: "module", Required: true},
		{Name: "providers", Required: false},
		{Name: "depends_on", Required: false},
		{Name: "skip_step", Required: false},
		{Name: "verifies", Required: false},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: blockTypeVariables},
	},
}

// ScenarioStep is a step in an Enos scenario.
type ScenarioStep struct {
	Name        string
	Description string
	Module      *Module
	Providers   map[string]*Provider
	DependsOn   []string
	Verifies    []*Quality
	Skip        bool
}

// NewScenarioStep returns a new Scenario step.
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
	diags := hcl.Diagnostics{}

	// Decode the our scenario step
	content, moreDiags := block.Body.Content(scenarioStepSchema)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	// Decode our name
	ss.Name = block.Labels[0]

	// Decode our description
	desc, ok := content.Attributes["description"]
	if ok {
		val, moreDiags := desc.Expr.Value(ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			return diags
		}
		ss.Description = val.AsString()
	}

	// Decode skip
	moreDiags, shouldSkip := ss.decodeSkip(content, ctx)
	diags = diags.Extend(moreDiags)
	ss.Skip = shouldSkip
	if moreDiags != nil && moreDiags.HasErrors() || shouldSkip {
		return diags
	}

	// Decode depends_on
	moreDiags = ss.decodeAndValidateDependsOn(content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	// Decode verifies
	moreDiags = ss.decodeAndValidateVerifies(content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	// Decode the step module reference
	moduleAttr, moreDiags := ss.decodeModuleAttribute(block, content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	// Validate that our module references an existing module in the eval context.
	moduleVal, moreDiags := ss.validateModuleAttributeReference(moduleAttr, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	// Handle our named providers, if any
	moreDiags = ss.decodeAndValidateProvidersAttribute(content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
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

// decodeSkip decodes the "skip_step" attribute and returns a boolean and diagnostics of whether
// or not the step should be skipped.
func (ss *ScenarioStep) decodeSkip(
	content *hcl.BodyContent,
	ctx *hcl.EvalContext,
) (
	hcl.Diagnostics,
	bool,
) {
	diags := hcl.Diagnostics{}

	skip, ok := content.Attributes["skip_step"]
	if !ok {
		return diags, false
	}

	val, moreDiags := skip.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags, false
	}

	if val.IsNull() || !val.IsWhollyKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "skip_step must be a known value",
			Subject:  skip.Expr.Range().Ptr(),
		})
	}

	if val.Type() != cty.Bool {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "skip_step must be a bool",
			Detail:   "skip_step must be a bool, not " + val.Type().FriendlyName(),
			Subject:  skip.Expr.Range().Ptr(),
		})
	}

	if diags.HasErrors() {
		return diags, false
	}

	return diags, val.True()
}

// decodeModuleAttribute decodes the module attribute from the content and ensures
// that it has the required source and name fields. It returns the HCL attribute
// for further validation later.
func (ss *ScenarioStep) decodeModuleAttribute(
	block *hcl.Block,
	content *hcl.BodyContent,
	ctx *hcl.EvalContext,
) (
	*hcl.Attribute,
	hcl.Diagnostics,
) {
	diags := hcl.Diagnostics{}

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
	if moreDiags != nil && moreDiags.HasErrors() {
		return module, diags
	}

	// Since we don't know the variable schema of the Terraform module we're
	// referencing we have to manually decode the parts that we do know. Everything
	// else we'll pass along to Terraform.
	if val.IsNull() || !val.IsWhollyKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid module value",
			Detail:   "module must be a known module object",
			Subject:  module.Expr.Range().Ptr(),
			Context:  hcl.RangeBetween(module.NameRange, module.Expr.Range()).Ptr(),
		})

		return module, diags
	}

	if val.Type().Equals(cty.String) {
		// We have a string address reference to a module. Try and locate it in
		// the eval context and set the appropriate source.
		modules, err := findEvalContextVariable("module", ctx)
		if err != nil {
			return module, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unknown module",
				Detail:   "no modules have been defined",
				Subject:  module.Expr.Range().Ptr(),
				Context:  hcl.RangeBetween(module.NameRange, module.Expr.Range()).Ptr(),
			})
		}

		// Validate that our module configuration references an existing module.
		// We only care that the name and source match. All other variables and
		// attributes we'll carry over.
		mod, ok := modules.AsValueMap()[val.AsString()]
		if !ok {
			return module, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unknown module",
				Detail:   fmt.Sprintf("no modules with name %s have been defined", val.AsString()),
				Subject:  module.Expr.Range().Ptr(),
				Context:  hcl.RangeBetween(module.NameRange, module.Expr.Range()).Ptr(),
			})
		}

		// Set our value to the module we found via the name reference
		val = mod
	}

	// We've been given a module value
	if !val.CanIterateElements() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid module value",
			Detail:   "module must be a string name or module value",
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
	diags := hcl.Diagnostics{}
	var modules cty.Value
	var moduleVal cty.Value
	var err error

	newDiag := func(sum string, det string) *hcl.Diagnostic {
		d := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  sum,
			Detail:   det,
		}
		if module != nil {
			d.Subject = module.Expr.Range().Ptr()
			d.Context = hcl.RangeBetween(module.NameRange, module.Expr.Range()).Ptr()
		}

		return d
	}

	// Search through the eval context chain until we find a "modules" variable.
	modules, err = findEvalContextVariable("module", ctx)
	if err != nil {
		diags = diags.Append(newDiag(
			"unknown module",
			fmt.Sprintf("a module with name %s has not been defined", ss.Module.Name),
		))

		return moduleVal, diags
	}

	// Validate that our module configuration references an existing module.
	// We only care that the name and source match. All other variables and
	// attributes we'll carry over.
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
			diags = diags.Append(newDiag(
				"missing source",
				"module value does not contain a source",
			))

			break
		}

		if s := source.AsString(); s != ss.Module.Source {
			diags = diags.Append(newDiag(
				"module source doesn't match module definition",
				fmt.Sprintf("module source for module %s is %s, not %s", ss.Module.Name, s, ss.Module.Source),
			))

			break
		}

		moduleVal = mod

		break
	}

	if moduleVal.IsNull() {
		diags = diags.Append(newDiag(
			"unknown module",
			fmt.Sprintf("a module with name %s has not been defined", ss.Module.Name),
		))

		return moduleVal, diags
	}

	return moduleVal, diags
}

// decodeAndValidateDependsOn decodess the depends_on attribute and ensures that
// the values reference known steps.
func (ss *ScenarioStep) decodeAndValidateDependsOn(content *hcl.BodyContent, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	depends, ok := content.Attributes["depends_on"]
	if !ok {
		return diags
	}

	ss.DependsOn = []string{}
	dependsOnSet := map[string]struct{}{}

	dependsVal, moreDiags := depends.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	if dependsVal.IsNull() || !dependsVal.IsWhollyKnown() || !dependsVal.CanIterateElements() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "depends value must be a known object",
			Subject:  depends.Expr.Range().Ptr(),
			Context:  depends.Range.Ptr(),
		})
	}

	// Get our defined steps from the eval context
	definedSteps, err := findEvalContextVariable("step", ctx)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No prior steps have been defined. You cannot depend_on an undefined step",
			Detail:   err.Error(),
			Subject:  depends.Expr.Range().Ptr(),
			Context:  depends.Range.Ptr(),
		})
	}

	// For each depends_on, make sure a matching step is defined and
	// matches
	for _, depV := range dependsVal.AsValueSlice() {
		if depV.Type().Equals(cty.String) {
			depName := depV.AsString()
			// We've been given a string value for our dep so it must be
			// an address to a step. Make sure it's defined.
			_, ok := definedSteps.AsValueMap()[depName]
			if !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "step has not been defined",
					Detail:   fmt.Sprintf("cannot depend_on step %s as it has not been defined", depName),
					Subject:  depends.Expr.Range().Ptr(),
					Context:  depends.Range.Ptr(),
				})

				continue
			}

			if _, exists := dependsOnSet[depName]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "cannot depend on the same step more than once",
					Detail:   fmt.Sprintf("cannot depend_on step %s more than once", depName),
					Subject:  depends.Expr.Range().Ptr(),
					Context:  depends.Range.Ptr(),
				})

				continue
			}

			dependsOnSet[depName] = struct{}{}

			continue
		}

		// We've been given some other value. Make sure it's a references to an
		// existing step, which exists in the eval context as a module value.
		step := NewModule()
		err := step.FromCtyValue(depV)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "value of depends_on attribute is not a step",
				Detail:   err.Error(),
				Subject:  depends.Expr.Range().Ptr(),
				Context:  depends.Range.Ptr(),
			})

			continue
		}

		if _, exists := dependsOnSet[step.Name]; exists {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "cannot depend on the same step more than once",
				Detail:   fmt.Sprintf("cannot depend_on step %s more than once", step.Name),
				Subject:  depends.Expr.Range().Ptr(),
				Context:  depends.Range.Ptr(),
			})

			continue
		}

		dependsOnSet[step.Name] = struct{}{}

		continue
	}

	for name := range dependsOnSet {
		ss.DependsOn = append(ss.DependsOn, name)
	}
	sort.Strings(ss.DependsOn)

	return diags
}

// decodeAndValidateVerifies decodess the verifies attribute. This attribute is one-or-more Verifies
// that have either been defined at the top level and should therefore be accessible in the eval context,
// or defined in-inline for singular qualities.
func (ss *ScenarioStep) decodeAndValidateVerifies(content *hcl.BodyContent, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	verifies, ok := content.Attributes["verifies"]
	if !ok {
		return diags
	}

	ss.Verifies = []*Quality{}

	verifiesVal, moreDiags := verifies.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	if verifiesVal.IsNull() || !verifiesVal.IsWhollyKnown() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "verifies value must be knowable at compile time",
			Subject:  verifies.Expr.Range().Ptr(),
			Context:  verifies.Range.Ptr(),
		})
	}

	// We allow configuring the verifies attribute as a single quality or a slice of qualities.
	// Handle the simple case first
	if !verifiesVal.CanIterateElements() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid input for 'verifies' attribute. Must be one-or-more qualities",
			Subject:  verifies.Expr.Range().Ptr(),
			Context:  verifies.Range.Ptr(),
		})

		return diags
	}

	if !verifiesVal.Type().IsTupleType() {
		quality := NewQuality()
		err := quality.FromCtyValue(verifiesVal)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "invalid input for 'verifies' attribute. Must be one-or-more qualities",
				Detail:   err.Error(),
				Subject:  verifies.Expr.Range().Ptr(),
				Context:  verifies.Range.Ptr(),
			})
		}

		ss.Verifies = append(ss.Verifies, quality)

		return diags
	}

	// Iterate over the verifies attributes and decode them. Prevent the same quality from being
	// defined twice.
	verifiesSet := map[string]struct{}{}
	for _, vv := range verifiesVal.AsValueSlice() {
		quality := NewQuality()
		err := quality.FromCtyValue(vv)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "invalid input for 'verifies' attribute. Must be one-or-more qualities",
				Detail:   err.Error(),
				Subject:  verifies.Expr.Range().Ptr(),
				Context:  verifies.Range.Ptr(),
			})

			continue
		}

		if _, exists := verifiesSet[quality.Name]; exists {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Detail:   "cannot define the same quality more than once",
				Summary:  "quality verification of " + quality.Name + " cannot be specified more than once",
				Subject:  verifies.Expr.Range().Ptr(),
				Context:  verifies.Range.Ptr(),
			})

			continue
		}

		verifiesSet[quality.Name] = struct{}{}
		ss.Verifies = append(ss.Verifies, quality)
	}

	slices.SortStableFunc(ss.Verifies, CompareQuality)

	return diags
}

// decodeAndValidateProvidersAttribute decodess the providers attribute
// from the content and validates that each sub-attribute references a defined
// provider.
func (ss *ScenarioStep) decodeAndValidateProvidersAttribute(content *hcl.BodyContent, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	providers, ok := content.Attributes["providers"]
	if !ok {
		return diags
	}

	ss.Providers = map[string]*Provider{}

	providersVal, moreDiags := providers.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
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

	// findProvider finds an unrolled provider given a provider type and alias
	findProvider := func(pType string, pAlias string) (cty.Value, hcl.Diagnostics) {
		diags := hcl.Diagnostics{}

		types, ok := unrolled[pType]
		if !ok {
			return cty.NilVal, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("provider type %s is not defined", pType),
				Subject:  providers.Expr.Range().Ptr(),
				Context:  providers.Range.Ptr(),
			})
		}

		alias, ok := types[pAlias]
		if !ok {
			return cty.NilVal, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("alias %s for provider type %s is not defined", pAlias, pType),
				Subject:  providers.Expr.Range().Ptr(),
				Context:  providers.Range.Ptr(),
			})
		}

		return alias, diags
	}

	// For each defined provider, make sure a matching instance is defined and
	// matches
	for providerImportName, providerVal := range providersVal.AsValueMap() {
		provider := NewProvider()

		if providerVal.Type().Equals(cty.String) {
			// We've been given a string value for our provider so it must be
			// an address. Break it apart and look for the corresponding value
			// to the address.
			parts := strings.Split(providerVal.AsString(), ".")
			if len(parts) != 2 {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "provider attribute must be a provider value or type.alias string",
					Detail:   fmt.Sprintf("provider value %s is not a valid provider address", providerVal.AsString()),
					Subject:  providers.Expr.Range().Ptr(),
					Context:  providers.Range.Ptr(),
				})

				continue
			}

			// Find a matching value in the eval context from our address
			var moreDiags hcl.Diagnostics
			providerVal, moreDiags = findProvider(parts[0], parts[1])
			diags = diags.Extend(moreDiags)
			if moreDiags != nil && moreDiags.HasErrors() {
				continue
			}

			// Make sure we're not importing a "default" provider since they
			// don't have aliases and thus cannot be used as a provider alias
			// value. It is also not necessary as the provider will be rendered
			// as the default with no alias and thus the step/module will use
			// it by default.
			if parts[1] == "default" {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "cannot use default provider as alias value",
					Detail:   fmt.Sprintf(`provider alias %s has been set to %s.%s. Providers that have been labeled as "default" cannot be used as aliases because they are always available to scenario`, providerImportName, parts[0], parts[1]),
					Subject:  providers.Expr.Range().Ptr(),
					Context:  providers.Range.Ptr(),
				})

				continue
			}

			// Marshal our provider value into our instance and add it to the providers
			// list.
			err := provider.FromCtyValue(providerVal)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "unable to unmarshal provider value",
					Detail:   err.Error(),
					Subject:  providers.Expr.Range().Ptr(),
					Context:  providers.Range.Ptr(),
				})

				continue
			}

			ss.Providers[providerImportName] = provider

			continue
		}

		err := provider.FromCtyValue(providerVal)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unable to unmarshal provider value for " + providerImportName,
				Detail:   err.Error(),
				Subject:  providers.Expr.Range().Ptr(),
				Context:  providers.Range.Ptr(),
			})

			continue
		}

		alias, moreDiags := findProvider(provider.Type, provider.Alias)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		// Make sure we're not importing a "default" provider since they
		// don't have aliases and thus cannot be used as a provider alias
		// value. It is also not necessary as the provider will be rendered
		// as the default with no alias and thus the step/module will use
		// it by default.
		if provider.Alias == "default" {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "cannot use default provider as alias value",
				Detail:   fmt.Sprintf(`provider alias %s has been set to %s.%s. Providers that have been labeled as "default" cannot be used as aliases because they are always available to scenario`, providerImportName, provider.Type, provider.Alias),
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

		ss.Module.Attrs[name] = StepVariableVal(&StepVariable{
			Value: value,
		})
	}
}

func (ss *ScenarioStep) decodeVariables(varBlocks hcl.Blocks, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	for _, varBlock := range varBlocks {
		// Step variables are decoded into special StepVariableType's because
		// they can be either known values or traversal references to previous
		// step outputs, which unknown to enos since it is not aware of the Terraform
		// module schema. Here, we will dynamically compose and HCL specification
		// for each variable in the variables block and then decode using our
		// special variable type.
		spec := hcldec.ObjectSpec{}
		attrs, moreDiags := varBlock.Body.JustAttributes()
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			return diags
		}

		attrs, moreDiags = filterTerraformMetaAttrs(attrs)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			return diags
		}

		for _, attr := range attrs {
			spec[attr.Name] = &hcldec.AttrSpec{
				Name:     attr.Name,
				Type:     StepVariableType,
				Required: true,
			}
		}

		val, moreDiags := hcldec.Decode(varBlock.Body, spec, ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		for attrName, attrVal := range val.AsValueMap() {
			ss.Module.Attrs[attrName] = attrVal
		}
	}

	return diags
}

// insertIntoCtx takes a pointer to an eval context and adds the step into
// it. If no "step" variable is present it will handle creating it.
func (ss *ScenarioStep) insertIntoCtx(ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}
	if ss == nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `cannot insert into nil "step" eval context`,
		})
	}

	var notDefined *errNotDefinedInCtx

	var steps map[string]cty.Value
	stepVal, err := findEvalContextVariable("step", ctx)
	if err != nil && !errors.As(err, &notDefined) {
		// This should never happen but lets make sure that it's not a different
		// error than we expect.
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `failed to search for "step" eval context`,
			Detail:   err.Error(),
		})
	}
	if err == nil {
		steps = stepVal.AsValueMap()
	}
	if steps == nil {
		steps = map[string]cty.Value{}
	}

	vals := map[string]cty.Value{
		"source":    cty.StringVal(ss.Module.Source),
		"name":      cty.StringVal(ss.Name),
		"variables": cty.ObjectVal(ss.Module.Attrs),
	}
	if ss.Module.Version != "" {
		vals["version"] = cty.StringVal(ss.Module.Version)
	}

	steps[ss.Name] = cty.ObjectVal(vals)
	if ctx.Variables == nil {
		ctx.Variables = map[string]cty.Value{}
	}
	ctx.Variables["step"] = cty.ObjectVal(steps)

	return diags
}

func (ss *ScenarioStep) outline() *pb.Scenario_Outline_Step {
	if ss == nil {
		return nil
	}

	out := &pb.Scenario_Outline_Step{
		Name:        ss.Name,
		Description: ss.Description,
	}

	for _, qual := range ss.Verifies {
		out.Verifies = append(out.GetVerifies(), qual.ToProto())
	}

	return out
}
