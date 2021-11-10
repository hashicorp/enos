package flightplan

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
)

var scenarioStepSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "module", Required: true},
	},
	Blocks: []hcl.BlockHeaderSchema{
		// NOTE: we don't currently handle variables block
		{Type: "variables", LabelNames: []string{"name"}},
	},
}

// ScenarioStep is a step in an Enos scenario
type ScenarioStep struct {
	Name   string
	Module *ScenarioStepModule
}

// ScenarioStepModule is the module attribute value in a scenario step
type ScenarioStepModule struct {
	Name   string
	Source string
	// NOTE: we don't currently handle the extra attributes
}

// NewScenarioStep returns a new Scenario step
func NewScenarioStep() *ScenarioStep {
	return &ScenarioStep{
		Module: &ScenarioStepModule{},
	}
}

// decode takes an HCL block and eval context and decodes itself from the block.
// It's responsible for ensuring that the block contains a "module" attribute
// and that it references an existing module that has been previously defined.
// It performs module reference validation by comparing our module reference
// to defined modules that are available in the eval context variable"module".
func (ss *ScenarioStep) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// Decode the our scenario step
	content, moreDiags := block.Body.Content(scenarioStepSchema)
	diags = diags.Extend(moreDiags)

	ss.Name = block.Labels[0]

	// Decode and validate module attribute
	module, ok := content.Attributes["module"]
	if !ok {
		r := block.Body.MissingItemRange()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "scenario step missing module",
			Detail:   "scenario step missing module",
			Subject:  &r,
		})

		return diags
	}

	// Now that we've found our module attribute we need to decode it and
	// validate that it's referring to an allowed module, i.e. one that has
	// been defined at the top-level and has a matching name and source.
	val, moreDiags := module.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	// Since we don't know the entire schema of the module we cannot use gocty
	// style struct decoding. We'll have to manually decode the parts that we
	// do know.
	if val.IsNull() || !val.IsWhollyKnown() || !val.CanIterateElements() {
		r := module.Expr.Range()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid module value",
			Detail:   "module must be a known module object",
			Subject:  &r,
		})

		return diags
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

		return diags
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

		return diags
	}
	ss.Module.Source = source.AsString()

	// Now that we know what module we're referencing, make sure that it references
	// a module that has been defined. We'll do this by going through our eval
	// context chain until the "module" variable is available. We'll decode it
	// and try to find a matching module.
	var modules cty.Value
	var foundModules bool
	for modCtx := ctx; modCtx != nil; modCtx = modCtx.Parent() {
		if modCtx == nil {
			r := module.Expr.Range()
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unknown module",
				Detail:   fmt.Sprintf("a module with name %s has not been defined", ss.Module.Name),
				Subject:  &r,
			})

			return diags
		}

		// Search through all of the eval contexts until we find "modules"
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

		return diags
	}

	foundModuleInCtx := false
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

		foundModuleInCtx = true
		break
	}

	if !foundModuleInCtx {
		r := module.Expr.Range()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unknown module",
			Detail:   fmt.Sprintf("a module with name %s has not been defined", ss.Module.Name),
			Subject:  &r,
		})

		return diags
	}

	return diags
}
