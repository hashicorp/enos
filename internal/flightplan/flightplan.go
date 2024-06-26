// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

const (
	attrLabelNameDefault = "name"
	attrLabelNameType    = "type"
	attrLabelNameAlias   = "alias"

	blockTypeBackend           = "backend"
	blockTypeCloud             = "cloud"
	blockTypeMatrixExclude     = "exclude"
	blockTypeGlobals           = "globals"
	blockTypeMatrixInclude     = "include"
	blockTypeLocals            = "locals"
	blockTypeMatrix            = "matrix"
	blockTypeModule            = "module"
	blockTypeOutput            = "output"
	blockTypeProvider          = "provider"
	blockTypeProviderMeta      = "provider_meta"
	blockTypeQuality           = "quality"
	blockTypeRequiredProviders = "required_providers"
	blockTypeSample            = "sample"
	blockTypeSampleSubset      = "subset"
	blockTypeScenario          = "scenario"
	blockTypeScenarioStep      = "step"
	blockTypeTerraformSetting  = "terraform"
	blockTypeTerraformCLI      = "terraform_cli"
	blockTypeValidation        = "validation"
	blockTypeVariable          = "variable"
	blockTypeVariables         = "variables"
)

var flightPlanSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: blockTypeGlobals},
		{Type: blockTypeSample, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeTerraformSetting, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeTerraformCLI, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeProvider, LabelNames: []string{attrLabelNameType, attrLabelNameAlias}},
		{Type: blockTypeQuality, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeScenario, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeModule, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeVariable, LabelNames: []string{attrLabelNameDefault}},
	},
}

// NewFlightPlan returns a new instance of a FlightPlan.
func NewFlightPlan(opts ...Opt) (*FlightPlan, error) {
	fp := &FlightPlan{
		Files:             map[string]*hcl.File{},
		Samples:           []*Sample{},
		TerraformSettings: []*TerraformSetting{},
		TerraformCLIs:     []*TerraformCLI{},
		Providers:         []*Provider{},
		ScenarioBlocks:    ScenarioBlocks{},
		Modules:           []*Module{},
	}

	for _, opt := range opts {
		err := opt(fp)
		if err != nil {
			return fp, err
		}
	}

	return fp, nil
}

// WithFlightPlanBaseDirectory sets the base directory to the absolute path
// of the directory given.
func WithFlightPlanBaseDirectory(dir string) Opt {
	return func(fp *FlightPlan) error {
		var err error
		fp.BaseDir, err = filepath.Abs(dir)

		return err
	}
}

// Opt is a flight plan option.
type Opt func(*FlightPlan) error

// FlightPlan represents our flight plan, the main configuration of Enos.
type FlightPlan struct {
	BaseDir           string
	BodyContent       *hcl.BodyContent
	Files             map[string]*hcl.File
	Modules           []*Module
	Providers         []*Provider
	Qualities         []*Quality
	TerraformSettings []*TerraformSetting
	TerraformCLIs     []*TerraformCLI
	Samples           []*Sample
	ScenarioBlocks    ScenarioBlocks
}

func (fp *FlightPlan) Scenarios() []*Scenario {
	if fp == nil || fp.ScenarioBlocks == nil || len(fp.ScenarioBlocks) < 1 {
		return nil
	}

	return fp.ScenarioBlocks.Scenarios()
}

// decodeVariables decodes "variable" blocks that are defined in the
// top-level schema and sets/validates values that might have been passed
// in via enos.vars.hcl or ENOS_VAR_ environment variables.
func (fp *FlightPlan) decodeVariables(
	ctx *hcl.EvalContext,
	varFiles map[string]*hcl.File,
	envVars []string,
) hcl.Diagnostics {
	diags := hcl.Diagnostics{}
	values := map[string]*VariableValue{}
	vars := map[string]cty.Value{}

	// Create a unified body for our user supplied variables
	files := []*hcl.File{}
	for _, file := range varFiles {
		files = append(files, file)
	}
	valuesBody := hcl.MergeFiles(files)

	// Do a sanity check to make sure people are not accidentally defining
	// "variable" blocks here instead of defining variable input values.
	content, _, _ := valuesBody.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       blockTypeVariable,
				LabelNames: []string{attrLabelNameDefault},
			},
		},
	})
	for _, block := range content.Blocks {
		name := block.Labels[0]
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Variable declaration in enos.vars.hcl file",
			Detail:   fmt.Sprintf("An enos.vars.hcl file is used to assign values to variables that have already been declared in enos.hcl files, not to declare new variables. To declare variable %q, place this block in one of your enos.hcl files, such as enos-variables.hcl.\n\nTo set a value for this variable in %s, use the definition syntax instead:\n    %s = <value>", name, block.TypeRange.Filename, name),
			Subject:  block.TypeRange.Ptr(),
		})
	}
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Extend(verifyNoBlockInAttrOnlySchema(valuesBody))
	if diags.HasErrors() {
		return diags
	}

	// Get the values of each variable
	vals, moreDiags := valuesBody.JustAttributes()
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	for _, val := range vals {
		values[val.Name] = &VariableValue{
			Expr:   val.Expr,
			Range:  val.Range,
			Source: VariableValueSourceVarsFile,
		}
	}

	// Now set any values that have been set from env vars. We do this last to
	// ensure that environment variables have the highest precedence.
	for _, envVar := range envVars {
		if !strings.HasPrefix(envVar, EnvVarPrefix) {
			continue
		}

		trimmed := envVar[len(EnvVarPrefix):]
		idx := strings.Index(trimmed, "=")
		if idx == -1 {
			continue
		}

		values[trimmed[:idx]] = &VariableValue{
			EnvVarRaw: trimmed[idx+1:],
			Source:    VariableValueSourceEnvVar,
			Range: hcl.Range{
				Filename: "environment_variables",
				Start:    hcl.InitialPos,
				End:      hcl.InitialPos,
			},
		}
	}

	// Now that we have our user-supplied variable values, we'll decode all of
	// the "variable" blocks in the normal flightplan config and set the values
	// as appropriate.
	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeVariable) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		variable := NewVariable()
		moreDiags = variable.decode(block, values)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		vars[variable.Name] = variable.Value()
	}

	// NOTE: We only keep track of variable values in the eval context. That is
	// fine for now but if we ever want to handle things like "sensitive" we'll
	// have to keep them around in the flight plan.
	ctx.Variables["var"] = cty.ObjectVal(vars)

	return diags
}

// decodeGlobals decodes "global" blocks that are defined in the top-level schema.
func (fp *FlightPlan) decodeGlobals(ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	globals := map[string]cty.Value{}
	for i, globalsBlock := range fp.BodyContent.Blocks.OfType(blockTypeGlobals) {
		if i == 0 {
			if ctx.Variables == nil {
				ctx.Variables = map[string]cty.Value{}
			}
		}

		moreDiags := verifyBlockHasNLabels(globalsBlock, 0)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		attrs, moreDiags := globalsBlock.Body.JustAttributes()
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		// Since our decoder gives us our globals as a map we cannot depend on them being in the
		// order in which they were defined. Rather than trying to topologically sort them by their
		// traversals, we'll sort them by their declared range offset. This requires scenario
		// authors to write locals in the order in which they are to be referred.
		sortedGlobals := []*hcl.Attribute{}
		for _, attr := range attrs {
			sortedGlobals = append(sortedGlobals, attr)
		}
		sort.Slice(sortedGlobals, func(i, j int) bool {
			return sortedGlobals[i].Range.Start.Byte < sortedGlobals[j].Range.Start.Byte
		})

		for _, attr := range sortedGlobals {
			val, moreDiags := attr.Expr.Value(ctx)
			diags = diags.Extend(moreDiags)
			if moreDiags != nil && moreDiags.HasErrors() {
				continue
			}

			globals[attr.Name] = val
			ctx.Variables["global"] = cty.ObjectVal(globals)
		}
	}

	return diags
}

// decodeSamples decodes "sample" blocks that are defined in the top-level schema.
func (fp *FlightPlan) decodeSamples(ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeSample) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		sample := NewSample()
		moreDiags = sample.Decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		fp.Samples = append(fp.Samples, sample)
	}

	return diags
}

// decodeTerraformSettings decodes "terraform" blocks that are defined in the
// top-level schema.
func (fp *FlightPlan) decodeTerraformSettings(ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}
	settings := map[string]cty.Value{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeTerraformSetting) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		setting := NewTerraformSetting()
		moreDiags = setting.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		fp.TerraformSettings = append(fp.TerraformSettings, setting)
		settings[setting.Name] = setting.ToCtyValue()
	}

	ctx.Variables["terraform"] = cty.ObjectVal(settings)

	return diags
}

// decodeQualities decodes "quality" blocks that are defined in the top-level schema.
func (fp *FlightPlan) decodeQualities(ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}
	qualities := map[string]cty.Value{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeQuality) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		quality := NewQuality()
		moreDiags = quality.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		_, previouslyDefined := qualities[quality.Name]
		if previouslyDefined {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "quality has previously been defined",
				Detail:   fmt.Sprintf(`quality %s has already been defined`, quality.Name),
				Subject:  block.DefRange.Ptr(),
			})

			continue
		}

		qualities[quality.Name] = quality.ToCtyValue()
		fp.Qualities = append(fp.Qualities, quality)
	}

	ctx.Variables["quality"] = cty.ObjectVal(qualities)

	return diags
}

// decodeTerraformCLIs decodes "terraform_cli" blocks that are defined in the
// top-level schema.
func (fp *FlightPlan) decodeTerraformCLIs(ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}
	clis := map[string]cty.Value{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeTerraformCLI) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		cli := NewTerraformCLI()
		moreDiags = cli.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		fp.TerraformCLIs = append(fp.TerraformCLIs, cli)
		clis[cli.Name] = cli.evalCtx()
	}

	if _, ok := clis["default"]; !ok {
		// Add the default terraform CLI if it has not been set
		d := DefaultTerraformCLI()
		fp.TerraformCLIs = append(fp.TerraformCLIs, d)
		clis["default"] = d.evalCtx()
	}

	ctx.Variables["terraform_cli"] = cty.ObjectVal(clis)

	return diags
}

// decodeProviders decodes "provider" blocks that are defined in the
// top-level schema.
func (fp *FlightPlan) decodeProviders(ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}
	// provider type -> alias name -> provider object value
	providers := map[string]map[string]cty.Value{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeProvider) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		moreDiags = verifyBlockHasNLabels(block, 2)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		provider := NewProvider()
		moreDiags = provider.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		fp.Providers = append(fp.Providers, provider)

		aliasesForType, ok := providers[provider.Type]
		if !ok {
			aliasesForType = map[string]cty.Value{}
		}
		_, previouslyDefined := aliasesForType[provider.Alias]
		if previouslyDefined {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "provider and alias has been previously defined",
				Detail:   fmt.Sprintf(`provider %s with alias %s has already been defined`, provider.Type, provider.Alias),
				Subject:  hcl.RangeBetween(block.LabelRanges[0], block.LabelRanges[1]).Ptr(),
				Context:  block.DefRange.Ptr(),
			})

			continue
		}

		aliasesForType[provider.Alias] = provider.ToCtyValue()
		providers[provider.Type] = aliasesForType
	}

	// Nest by type and alias so we can access it in the eval context
	// as providers.type.alias, eg: providers.aws.east.attrs.region
	vals := map[string]cty.Value{}
	for providerType, aliases := range providers {
		vals[providerType] = cty.ObjectVal(aliases)
	}
	ctx.Variables["provider"] = cty.ObjectVal(vals)

	return diags
}

// decodeModules decodes "module" blocks that are defined in the top-level
// schema.
func (fp *FlightPlan) decodeModules(ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}
	mods := map[string]cty.Value{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeModule) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		module := NewModule()
		moreDiags = module.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		fp.Modules = append(fp.Modules, module)
		mods[module.Name] = module.ToCtyValue()
	}

	ctx.Variables["module"] = cty.ObjectVal(mods)

	return diags
}

// decodeMatrix takes an eval context and scenario blocks and decodes only the
// matrix block. It returns a unique matrix with vectors for all unique variant
// value combinations.
func decodeMatrix(ctx *hcl.EvalContext, block *hcl.Block) (*MatrixBlock, hcl.Diagnostics) {
	decoder := newMatrixDecoder()
	return decoder.decodeMatrix(ctx, block)
}

// filterTerraformMetaAttrs does our best to ensure that the given set of
// attributes does not include unsupported Terraform meta-args. This is intended
// to handle cases where we don't know the underlying schema of a blocks attributes
// is unknown or flexible (e.g. module or scenario step variables) but we still
// want to make sure that users haven't passed in disallowed meta-args such as
// count, for_each, and depends_on. Here we'll filter out bad attributes and
// pass along error diagnostics for any encountered. This allows us to continue
// to decode in some cases while still bubbling the error up.
func filterTerraformMetaAttrs(in hcl.Attributes) (hcl.Attributes, hcl.Diagnostics) {
	diags := hcl.Diagnostics{}
	var out hcl.Attributes

	for name, attr := range in {
		switch attr.Name {
		case "count", "for_each", "depends_on":
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "invalid attribute",
				Detail:   fmt.Sprintf(`Terraform meta-arguments "%s" are not valid`, attr.Name),
				Subject:  attr.NameRange.Ptr(),
				Context:  hcl.RangeBetween(attr.NameRange, attr.Range).Ptr(),
			})
		default:
			if out == nil {
				out = hcl.Attributes{}
			}
			out[name] = attr
		}
	}

	return out, diags
}

// verifyNoBlockInAttrOnlySchema is a hacky way to ensure that the given block
// doesn't have any child blocks. As we often have to deal with attribute only
// blocks which have an unknown schema, this is the only way to ensure those
// blocks don't have child blocks.
func verifyNoBlockInAttrOnlySchema(in hcl.Body) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	body, ok := in.(*hclsyntax.Body)
	if ok && len(body.Blocks) != 0 {
		for _, block := range body.Blocks {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unexpected block",
				Subject:  block.TypeRange.Ptr(),
				Context:  hcl.RangeBetween(block.TypeRange, block.Range()).Ptr(),
			})
		}
	}

	return diags
}

// verifyBodyOnlyHasBlocksWithLabels is a hacky way to ensure that the given block
// doesn't have any child blocks execept for those allowed.
func verifyBodyOnlyHasBlocksWithLabels(in hcl.Body, allowed ...string) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	body, ok := in.(*hclsyntax.Body)
	if ok && len(body.Blocks) != 0 {
		for _, block := range body.Blocks {
			isAllowed := false
			for _, allowed := range allowed {
				if block.Type == allowed {
					isAllowed = true

					break
				}
			}
			if !isAllowed {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "unexpected block",
					Detail:   fmt.Sprintf("block type %s is not allowed, must be one of %x", block.Type, allowed),
					Subject:  block.TypeRange.Ptr(),
					Context:  hcl.RangeBetween(block.TypeRange, block.Range()).Ptr(),
				})
			}
		}
	}

	return diags
}

// verifyBlockLabelsAreValidIdentifiers takes and HCL block and validates that
// the labels conform to both HCL and Enos allowed identifiers.
func verifyBlockLabelsAreValidIdentifiers(block *hcl.Block) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	if len(block.Labels) == 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid block identifier",
			Detail:   "block can only have a single name label",
			Subject:  block.TypeRange.Ptr(),
			Context:  hcl.RangeBetween(block.TypeRange, block.DefRange).Ptr(),
		})

		return diags
	}

	for i, label := range block.Labels {
		diags = diags.Extend(verifyValidIdentifier(label, block.LabelRanges[i].Ptr()))
	}

	return diags
}

// verifyValidIdentifier verifies that the string value could be used as an enos identifier.
func verifyValidIdentifier(label string, hclRange *hcl.Range) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	// Make sure it's a valid HCL identifier
	if !hclsyntax.ValidIdentifier(label) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "label is invalid",
			Detail:   "label is not a valid HCL identifier",
			Subject:  hclRange,
		})
	}

	// Make sure it also adheres to Enos block name rules
	r := regexp.MustCompile(`^[\w]+$`)
	if !r.MatchString(label) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "label is invalid",
			Detail:   "label is not a valid enos identifier",
			Subject:  hclRange,
		})
	}

	return diags
}

// verifyBlockHasNLabels verifies that the given block has the appropriate number
// of defined labels.
func verifyBlockHasNLabels(block *hcl.Block, n int) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	if len(block.Labels) != n {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid block",
			Detail:   fmt.Sprintf("block has %d labels but required %d", len(block.Labels), n),
			Subject:  block.TypeRange.Ptr(),
			Context:  block.DefRange.Ptr(),
		})
	}

	return diags
}
