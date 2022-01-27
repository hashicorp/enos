package flightplan

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var scenarioSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "terraform_cli", Required: false},
		{Name: "terraform", Required: false},
		{Name: "providers", Required: false},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: blockTypeScenarioStep, LabelNames: []string{"name"}},
	},
}

// Scenario represents an Enos scenario
type Scenario struct {
	Name             string
	TerraformCLI     *TerraformCLI
	TerraformSetting *TerraformSetting
	Steps            []*ScenarioStep
	Providers        []*Provider
}

// NewScenario returns a new Scenario
func NewScenario() *Scenario {
	return &Scenario{
		TerraformCLI: NewTerraformCLI(),
		Steps:        []*ScenarioStep{},
		Providers:    []*Provider{},
	}
}

// decode takes an HCL block and an evalutaion context and it decodes itself
// from the block. Any errors that are encountered during decoding will be
// returned as hcl diagnostics.
func (s *Scenario) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	content, moreDiags := block.Body.Content(scenarioSchema)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	s.Name = block.Labels[0]

	// Decode all of our blocks. Make sure that scenario has at least one
	// step.
	foundSteps := map[string]struct{}{}
	for _, childBlock := range content.Blocks {
		switch childBlock.Type {
		case blockTypeScenarioStep:
			if _, dupeStep := foundSteps[childBlock.Labels[0]]; dupeStep {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "redeclared step in scenario",
					Detail:   fmt.Sprintf("a step with name %s has already been declared", childBlock.Labels[0]),
					Subject:  childBlock.TypeRange.Ptr(),
					Context:  hcl.RangeBetween(childBlock.TypeRange, childBlock.DefRange).Ptr(),
				})
				continue
			}

			moreDiags = verifyBlockLabelsAreValidIdentifiers(childBlock)
			diags = diags.Extend(moreDiags)
			if moreDiags.HasErrors() {
				continue
			}

			step := NewScenarioStep()
			moreDiags = step.decode(childBlock, ctx)
			diags = diags.Extend(moreDiags)
			if moreDiags.HasErrors() {
				continue
			}

			foundSteps[step.Name] = struct{}{}
			s.Steps = append(s.Steps, step)
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unknown block in scenario",
				Detail:   fmt.Sprintf(`unable to parse unknown block "%s" in scenario`, childBlock.Type),
				Subject:  childBlock.TypeRange.Ptr(),
				Context:  hcl.RangeBetween(childBlock.TypeRange, childBlock.DefRange).Ptr(),
			})
		}
	}

	if len(foundSteps) == 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "missing required step block",
			Detail:   "scenarios require one or more step blocks",
			Subject:  block.Body.MissingItemRange().Ptr(),
		})
	}

	// Decode the scenario terraform_cli reference
	moreDiags = s.decodeAndValidateTerraformCLIAttribute(content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	// Decode the scenario terraform reference
	moreDiags = s.decodeAndValidateTerraformSettingsAttribute(content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	// Decode the scenario providers
	moreDiags = s.decodeAndValidateProvidersAttribute(content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	return diags
}

// decodeAndValidateTerraformCLIAttribute decodess the terraform_cli attribute
// from the content and validates that it refers to an existing terraform_cli.
func (s *Scenario) decodeAndValidateTerraformCLIAttribute(
	content *hcl.BodyContent,
	ctx *hcl.EvalContext,
) hcl.Diagnostics {
	var diags hcl.Diagnostics

	terraformCli, ok := content.Attributes["terraform_cli"]
	if !ok {
		// The terraform_cli attribute has not been set so we'll use the default
		// terraform_cli which we'll get from the eval context.
		diag := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to determine terraform_cli",
			Detail:   "no default terraform_cli's are available in the eval ctx",
			Subject:  content.MissingItemRange.Ptr(),
		}

		terraformClis, err := findEvalContextVariable("terraform_cli", ctx)
		if err != nil {
			return diags.Append(diag)
		}

		defaultCli, ok := terraformClis.AsValueMap()["default"]
		if !ok {
			return diags.Append(diag)
		}

		err = gocty.FromCtyValue(defaultCli, &s.TerraformCLI)
		if err != nil {
			diag.Summary = "unable to convert default terraform_cli from eval context to object"
			diag.Detail = err.Error()
			return diags.Append(diag)
		}

		return diags
	}

	// Decode our terraform_cli from the eval context. If it hasn't been defined
	// this will raise an error.
	moreDiags := gohcl.DecodeExpression(terraformCli.Expr, ctx, &s.TerraformCLI)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	return diags
}

// decodeAndValidateTerraformSettingsAttribute decodess the terraform attribute
// from the content and validates that it refers to an existing terraform.
func (s *Scenario) decodeAndValidateTerraformSettingsAttribute(
	content *hcl.BodyContent,
	ctx *hcl.EvalContext,
) hcl.Diagnostics {
	var diags hcl.Diagnostics

	terraformSetting, ok := content.Attributes["terraform"]
	if ok {
		// A "terraform" attribute value has been set. Make sure it matches
		// one that has been defined in the outer scope.
		tfSettingsVal, moreDiags := terraformSetting.Expr.Value(ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			return diags
		}

		if tfSettingsVal.IsNull() || !tfSettingsVal.IsWhollyKnown() || !tfSettingsVal.CanIterateElements() {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "terraform value must be that of a terraform block",
				Subject:  terraformSetting.Expr.Range().Ptr(),
				Context:  terraformSetting.Range.Ptr(),
			})
		}

		// Find it in the eval context and make sure it matches
		terraformSettings, err := findEvalContextVariable("terraform", ctx)
		if err != nil {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "terraform references an undefined terraform block",
				Detail:   err.Error(),
				Subject:  terraformSetting.Expr.Range().Ptr(),
				Context:  terraformSetting.Range.Ptr(),
			})
		}

		settingsName, ok := tfSettingsVal.AsValueMap()["name"]
		if !ok {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "terraform value does not have the required name attribute",
				Subject:  terraformSetting.Expr.Range().Ptr(),
				Context:  terraformSetting.Range.Ptr(),
			})
		}

		if settingsName.IsNull() || !settingsName.IsWhollyKnown() {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "terraform name value must be known",
				Subject:  terraformSetting.Expr.Range().Ptr(),
				Context:  terraformSetting.Range.Ptr(),
			})
		}

		setting, ok := terraformSettings.AsValueMap()[settingsName.AsString()]
		if !ok {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "terraform references an undefined terraform block",
				Detail:   fmt.Sprintf("no terraform block with a name label %s exists", settingsName.AsString()),
				Subject:  terraformSetting.Expr.Range().Ptr(),
				Context:  terraformSetting.Range.Ptr(),
			})
		}

		if tfSettingsVal.Equals(setting) != cty.True {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "terraform value and configured value don't match",
				Subject:  terraformSetting.Expr.Range().Ptr(),
				Context:  terraformSetting.Range.Ptr(),
			})
		}

		s.TerraformSetting = NewTerraformSetting()
		err = s.TerraformSetting.FromCtyValue(setting)
		if err != nil {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unable to unmarshal terraform from eval context",
				Detail:   err.Error(),
				Subject:  terraformSetting.Expr.Range().Ptr(),
				Context:  terraformSetting.Range.Ptr(),
			})
		}

		return diags
	}

	// The terraform attribute has not been set so we'll use the default
	// terraform settings if they exist.
	terraformSettings, err := findEvalContextVariable("terraform", ctx)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "terraform references an undefined terraform block",
			Detail:   err.Error(),
			Subject:  terraformSetting.Expr.Range().Ptr(),
			Context:  terraformSetting.Range.Ptr(),
		})
	}

	setting, ok := terraformSettings.AsValueMap()["default"]
	if !ok {
		return diags
	}

	s.TerraformSetting = NewTerraformSetting()
	err = s.TerraformSetting.FromCtyValue(setting)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to unmarshal terraform from eval context",
			Detail:   err.Error(),
			Subject:  terraformSetting.Expr.Range().Ptr(),
			Context:  terraformSetting.Range.Ptr(),
		})
	}

	return diags
}

// decodeAndValidateProvidersAttribute decodess the providers attribute
// from the content and validates that each sub-attribute references a defined
// provider.
func (s *Scenario) decodeAndValidateProvidersAttribute(
	content *hcl.BodyContent,
	ctx *hcl.EvalContext,
) hcl.Diagnostics {
	var diags hcl.Diagnostics

	providers, ok := content.Attributes["providers"]
	if !ok {
		return diags
	}

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
	for _, providerVal := range providersVal.AsValueSlice() {
		provider := NewProvider()
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

		s.Providers = append(s.Providers, provider)
	}

	return diags
}
