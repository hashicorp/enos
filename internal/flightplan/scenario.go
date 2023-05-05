package flightplan

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
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
		{Type: blockTypeScenarioStep, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeOutput, LabelNames: []string{attrLabelNameDefault}},
		// Matrix blocks are handled by the matrix decoder
		{Type: blockTypeMatrix},
		{Type: blockTypeLocals},
	},
}

// Scenario represents an Enos scenario
type Scenario struct {
	Name             string
	Variants         *Vector
	TerraformCLI     *TerraformCLI
	TerraformSetting *TerraformSetting
	Steps            []*ScenarioStep
	Providers        []*Provider
	Outputs          []*ScenarioOutput
}

// NewScenario returns a new Scenario
func NewScenario() *Scenario {
	return &Scenario{
		TerraformCLI: NewTerraformCLI(),
		Steps:        []*ScenarioStep{},
		Providers:    []*Provider{},
		Outputs:      []*ScenarioOutput{},
	}
}

// String returns the scenario identifiers as a string
func (s *Scenario) String() string {
	str := s.Name
	if s.Variants != nil && len(s.Variants.elements) > 0 {
		str = fmt.Sprintf("%s %s", str, s.Variants.String())
	}

	return str
}

// UID returns a unique identifier from the name and variants
func (s *Scenario) UID() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s.String())))
}

// Ref returns the proto reference
func (s *Scenario) Ref() *pb.Ref_Scenario {
	return &pb.Ref_Scenario{
		Id: &pb.Scenario_ID{
			Name:     s.Name,
			Variants: s.Variants.Proto(),
			Uid:      s.UID(),
		},
	}
}

// FromRef takes a unmarshals a scenario reference into itself
func (s *Scenario) FromRef(ref *pb.Ref_Scenario) {
	if ref == nil {
		return
	}

	s.Name = ref.GetId().GetName()
	s.Variants = NewVectorFromProto(ref.GetId().GetVariants())
}

// Match takes a filter and determines whether or not the scenario matches
// it.
func (s *Scenario) Match(filter *ScenarioFilter) bool {
	if filter == nil {
		return false
	}

	if filter.SelectAll {
		return true
	}

	// Get scenarios that match our name
	if filter.Name != "" && filter.Name != s.Name {
		return false
	}

	// If our scenario doesn't have any variants make make sure we don't have
	// a filter with includes or excludes.
	if s.Variants == nil || len(s.Variants.elements) == 0 {
		if filter.Include != nil && len(filter.Include.elements) > 0 {
			return false
		}

		if filter.Exclude != nil && len(filter.Exclude) > 0 {
			return false
		}
	}

	// Make sure it matches any includes
	if filter.Include != nil && len(filter.Include.elements) > 0 {
		if !s.Variants.ContainsUnordered(filter.Include) {
			return false
		}
	}

	// Make sure it does not match an exclude
	for _, ex := range filter.Exclude {
		if ex.Match(s.Variants) {
			return false
		}
	}

	return true
}

// decode takes an HCL block and an evalutaion context and it decodes itself
// from the block. Any errors that are encountered during decoding will be
// returned as hcl diagnostics.
func (s *Scenario) decode(block *hcl.Block, ctx *hcl.EvalContext, mode DecodeMode) hcl.Diagnostics {
	var diags hcl.Diagnostics

	s.Name = block.Labels[0]

	if mode == DecodeModeRef {
		return diags
	}

	content, moreDiags := block.Body.Content(scenarioSchema)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	// Make sure that scenario has at least one step.
	if len(content.Blocks.OfType(blockTypeScenarioStep)) < 1 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "missing required step block",
			Detail:   "scenarios require one or more step blocks",
			Subject:  block.Body.MissingItemRange().Ptr(),
		})
	}

	// Decode our locals
	moreDiags = s.decodeAndValidateLocalsBlock(content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
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

	// Decode all of our step blocks.
	moreDiags = s.decodeAndValidateStepBlocks(content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	// Decode our outputs
	moreDiags = s.decodeAndValidateOutputBlocks(content, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	return diags
}

// decodeAndValidateLocalsBlock decodes the locals block and makes the values
// available in the evaluation context of the scenario.
func (s *Scenario) decodeAndValidateLocalsBlock(
	content *hcl.BodyContent,
	ctx *hcl.EvalContext,
) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if len(content.Blocks.OfType(blockTypeLocals)) == 0 {
		return diags
	}

	locals := map[string]cty.Value{}
	for i, localsBlock := range content.Blocks.OfType(blockTypeLocals) {
		if i == 0 {
			if ctx.Variables == nil {
				ctx.Variables = map[string]cty.Value{}
			}
		}

		moreDiags := verifyBlockHasNLabels(localsBlock, 0)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		attrs, moreDiags := localsBlock.Body.JustAttributes()
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		// Since our decoder gives us our locals as a map we cannot depend
		// on them being in the order in which they were defined. Rather than
		// trying to topologically sort them by their traversals, we'll sort them
		// by their declared range offset. This requires scenario authors to
		// write locals in the order in which they are to be referred.
		sortedLocals := []*hcl.Attribute{}
		for _, attr := range attrs {
			sortedLocals = append(sortedLocals, attr)
		}
		sort.Slice(sortedLocals, func(i, j int) bool {
			return sortedLocals[i].Range.Start.Byte < sortedLocals[j].Range.Start.Byte
		})

		for _, attr := range sortedLocals {
			val, moreDiags := attr.Expr.Value(ctx)
			diags = diags.Extend(moreDiags)
			if moreDiags.HasErrors() {
				continue
			}

			locals[attr.Name] = val
			ctx.Variables["local"] = cty.ObjectVal(locals)
		}
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

	findAndLoadCLI := func(name string) hcl.Diagnostics {
		var diags hcl.Diagnostics

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

		cli, ok := terraformClis.AsValueMap()[name]
		if !ok {
			return diags.Append(diag)
		}

		err = gocty.FromCtyValue(cli, &s.TerraformCLI)
		if err != nil {
			diag.Summary = "unable to convert default terraform_cli from eval context to object"
			diag.Detail = err.Error()
			return diags.Append(diag)
		}

		return diags
	}

	terraformCli, ok := content.Attributes["terraform_cli"]
	if !ok {
		// The terraform_cli attribute has not been set so we'll use the default
		// terraform_cli which we'll get from the eval context.
		moreDiags := findAndLoadCLI("default")
		diags = diags.Extend(moreDiags)
		return diags
	}

	terraformCliVal, moreDiags := terraformCli.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	if terraformCliVal.Type().Equals(cty.String) {
		// Our value has been set to a string address.
		moreDiags := findAndLoadCLI(terraformCliVal.AsString())
		diags = diags.Extend(moreDiags)
		return diags

	}

	// Decode our terraform_cli from the eval context. If it hasn't been defined
	// this will raise an error.
	moreDiags = gohcl.DecodeExpression(terraformCli.Expr, ctx, &s.TerraformCLI)
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

	if !ok || terraformSetting == nil {
		// The terraform attribute has not been set so we'll use the default
		// terraform settings if they exist.
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

	// A "terraform" attribute value has been set. Make sure it matches
	// one that has been defined in the outer scope.
	tfSettingsVal, moreDiags := terraformSetting.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	if tfSettingsVal.IsNull() || !tfSettingsVal.IsWhollyKnown() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "terraform value must be set to a terraform block label or value",
			Subject:  terraformSetting.Expr.Range().Ptr(),
			Context:  terraformSetting.Range.Ptr(),
		})
	}

	// The value of the terraform settings attribute can be either a string
	// name of the terraform settings to use, or the exact value of a
	// terraform settings that has been defined. We'll handle both cases
	// here.

	if tfSettingsVal.Type().Equals(cty.String) {
		// They set the value to a string so we'll set it to the value of a
		// terraform settings in the eval context.
		setting, ok := terraformSettings.AsValueMap()[tfSettingsVal.AsString()]
		if !ok {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "terraform references an undefined terraform block",
				Detail:   fmt.Sprintf("no terraform block with a name label %s exists", tfSettingsVal.AsString()),
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

	// Okay, it's not a string, it must be an exact terraform settings value.
	if !tfSettingsVal.CanIterateElements() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "terraform value must be set to a terraform block label or value",
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

	// findProvider finds an unrolled provider given a provider type and alias
	findProvider := func(pType string, pAlias string) (cty.Value, hcl.Diagnostics) {
		var diags hcl.Diagnostics

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

	// For each provider value that has been given, make sure addressed providers
	// exist and that provider values match.
	for _, providerVal := range providersVal.AsValueSlice() {
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
			if moreDiags.HasErrors() {
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

			s.Providers = append(s.Providers, provider)
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

		// Our provider value must be an actual provider. Find the corresponding
		// unrolled provider and ensure that it matches.
		alias, moreDiags := findProvider(provider.Type, provider.Alias)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
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

func (s *Scenario) decodeAndValidateStepBlocks(
	content *hcl.BodyContent,
	ctx *hcl.EvalContext,
) hcl.Diagnostics {
	var diags hcl.Diagnostics
	foundSteps := map[string]struct{}{}

	for _, childBlock := range content.Blocks.OfType(blockTypeScenarioStep) {
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

		moreDiags := verifyBlockLabelsAreValidIdentifiers(childBlock)
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

		// update the eval context after each step is decoded. This way we can
		// make previously defined step's variables and module references available
		// to subsequent steps.
		moreDiags = step.insertIntoCtx(ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		foundSteps[step.Name] = struct{}{}
		s.Steps = append(s.Steps, step)
	}

	return diags
}

func (s *Scenario) decodeAndValidateOutputBlocks(
	content *hcl.BodyContent,
	ctx *hcl.EvalContext,
) hcl.Diagnostics {
	var diags hcl.Diagnostics
	foundOutputs := map[string]struct{}{}

	for _, childBlock := range content.Blocks.OfType(blockTypeOutput) {
		if _, dupeOut := foundOutputs[childBlock.Labels[0]]; dupeOut {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "redeclared output in scenario",
				Detail:   fmt.Sprintf("an output with name %s has already been declared", childBlock.Labels[0]),
				Subject:  childBlock.TypeRange.Ptr(),
				Context:  hcl.RangeBetween(childBlock.TypeRange, childBlock.DefRange).Ptr(),
			})
			continue
		}

		moreDiags := verifyBlockLabelsAreValidIdentifiers(childBlock)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		out := NewScenarioOutput()
		moreDiags = out.decode(childBlock, ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		foundOutputs[out.Name] = struct{}{}
		s.Outputs = append(s.Outputs, out)
	}

	return diags
}
