package flightplan

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
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
	blockTypeMatrixInclude     = "include"
	blockTypeLocals            = "locals"
	blockTypeMatrix            = "matrix"
	blockTypeModule            = "module"
	blockTypeOutput            = "output"
	blockTypeProvider          = "provider"
	blockTypeProviderMeta      = "provider_meta"
	blockTypeRequiredProviders = "required_providers"
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
		{Type: blockTypeTerraformSetting, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeTerraformCLI, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeProvider, LabelNames: []string{attrLabelNameType, attrLabelNameAlias}},
		{Type: blockTypeScenario, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeModule, LabelNames: []string{attrLabelNameDefault}},
		{Type: blockTypeVariable, LabelNames: []string{attrLabelNameDefault}},
	},
}

// NewFlightPlan returns a new instance of a FlightPlan
func NewFlightPlan(opts ...Opt) (*FlightPlan, error) {
	fp := &FlightPlan{
		Files:             map[string]*hcl.File{},
		TerraformSettings: []*TerraformSetting{},
		TerraformCLIs:     []*TerraformCLI{},
		Providers:         []*Provider{},
		Scenarios:         []*Scenario{},
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

// Opt is a flight plan option
type Opt func(*FlightPlan) error

// FlightPlan represents our flight plan, the main configuration of Enos.
type FlightPlan struct {
	BaseDir           string
	BodyContent       *hcl.BodyContent
	Files             map[string]*hcl.File
	TerraformSettings []*TerraformSetting
	TerraformCLIs     []*TerraformCLI
	Providers         []*Provider
	Scenarios         []*Scenario
	Modules           []*Module
}

// Decode takes a base eval content and HCL body and decodes it in chunks,
// continually expanding the evaluation context as more sub-sections are
// decoded. It returns HCL diagnostics that are collected over the course of
// decoding.
func (fp *FlightPlan) Decode(ctx *hcl.EvalContext, fpFiles, varsFiles map[string]*hcl.File) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if fp.BaseDir == "" {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to decode flight plan without a base directory",
		})
	}

	if ctx == nil {
		ctx = &hcl.EvalContext{
			Variables: map[string]cty.Value{},
			Functions: map[string]function.Function{},
		}
	}

	// Create a "unified" body of all flightplan files to use for decoding
	files := []*hcl.File{}
	for _, file := range fpFiles {
		files = append(files, file)
	}
	body := hcl.MergeFiles(files)

	// Decode our top-level schema
	fp.BodyContent, diags = body.Content(flightPlanSchema)

	// Decode and validate our variables. They'll be added to eval context for
	// access in later decoding.
	diags = diags.Extend(fp.decodeVariables(ctx, varsFiles))

	// Decode child blocks. Each child block decoder is responsible for
	// extending the evaluation context.
	diags = diags.Extend(fp.decodeTerraformSettings(ctx))
	diags = diags.Extend(fp.decodeTerraformCLIs(ctx))
	diags = diags.Extend(fp.decodeProviders(ctx))
	diags = diags.Extend(fp.decodeModules(ctx))
	diags = diags.Extend(fp.decodeScenarios(ctx))

	return diags
}

// decodeVariables decodes "variable" blocks that are defined in the
// top-level schema and sets/validates values that might have been passed
// in via enos.vars.hcl files.
func (fp *FlightPlan) decodeVariables(ctx *hcl.EvalContext, varFiles map[string]*hcl.File) hcl.Diagnostics {
	var diags hcl.Diagnostics
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
			Subject:  &block.TypeRange,
		})
	}
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Extend(verifyNoBlockInAttrOnlySchema(valuesBody))

	// Get the values of each variable
	vals, moreDiags := valuesBody.JustAttributes()
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	for _, val := range vals {
		ctyVal, moreDiags := val.Expr.Value(nil) // variable values don't have access to anything
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		values[val.Name] = &VariableValue{
			Value: ctyVal,
			Range: val.Range,
		}
	}

	// Now that we have our user-supplied variable values, we'll decode all of
	// the "variable" blocks in the normal flightplan config and set the values
	// as appropriate.
	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeVariable) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		variable := NewVariable()
		moreDiags = variable.decode(block, values)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		vars[variable.Name] = variable.Value()
	}

	// NOTE: At this point we're putting the variables into the eval context and
	// forgetting about their parsed values. That means we're losing "sensitive"
	// data and the like but that should be fine for now as we're not doing output
	// yet.
	ctx.Variables["var"] = cty.ObjectVal(vars)

	return diags
}

// decodeTerraformSettings decodes "terraform" blocks that are defined in the
// top-level schema.
func (fp *FlightPlan) decodeTerraformSettings(ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics
	settings := map[string]cty.Value{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeTerraformSetting) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		setting := NewTerraformSetting()
		moreDiags = setting.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		fp.TerraformSettings = append(fp.TerraformSettings, setting)
		settings[setting.Name] = setting.ToCtyValue()
	}

	ctx.Variables["terraform"] = cty.ObjectVal(settings)

	return diags
}

// decodeTerraformCLIs decodes "terraform_cli" blocks that are defined in the
// top-level schema.
func (fp *FlightPlan) decodeTerraformCLIs(ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics
	clis := map[string]cty.Value{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeTerraformCLI) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		cli := NewTerraformCLI()
		moreDiags = cli.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
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
	var diags hcl.Diagnostics
	// type -> map of aliases -> provider object
	providers := map[string]map[string]cty.Value{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeProvider) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		moreDiags = verifyBlockHasNLabels(block, 2)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		provider := NewProvider()
		moreDiags = provider.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
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
	var diags hcl.Diagnostics
	mods := map[string]cty.Value{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeModule) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		module := NewModule()
		moreDiags = module.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		fp.Modules = append(fp.Modules, module)
		mods[module.Name] = module.ToCtyValue()
	}

	ctx.Variables["module"] = cty.ObjectVal(mods)

	return diags
}

// decodeScenarios decodes the "scenario" blocks that are defined in the
// top-level schema.
func (fp *FlightPlan) decodeScenarios(ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeScenario) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		// Decode the matrix block and expand the scenarios.
		matrix, moreDiags := decodeMatrix(ctx, block)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		if matrix == nil || len(matrix.Vectors) == 0 {
			// Our scenario doesn't include a matrix
			scenario := NewScenario()
			moreDiags = scenario.decode(block, ctx.NewChild())
			diags = diags.Extend(moreDiags)
			if moreDiags.HasErrors() {
				return diags
			}
			fp.Scenarios = append(fp.Scenarios, scenario)
			continue
		}

		for _, vec := range matrix.Vectors {
			// Create a scenario for every matrix variant combination
			scenario := NewScenario()
			scenario.Variants = vec
			matrixCtx := ctx.NewChild()
			matrixCtx.Variables = map[string]cty.Value{
				"matrix": vec.CtyVal(),
			}
			moreDiags = scenario.decode(block, matrixCtx)
			diags = diags.Extend(moreDiags)
			if moreDiags.HasErrors() {
				// Exit early rather than continuing to decode all scenarios.
				// We do this to avoid spamming the output with identical
				// diagnostics for common errors that will be found in all
				// of the scenarios.
				return diags
			}

			fp.Scenarios = append(fp.Scenarios, scenario)
		}
	}

	sort.Slice(fp.Scenarios, func(i, j int) bool {
		return fp.Scenarios[i].String() < fp.Scenarios[j].String()
	})

	return diags
}

// decodeMatrix takes an eval context and scenario blocks and decodes only the
// matrix block. It returns a unique matrix with vectors for all unique variant
// value combinations.
func decodeMatrix(ctx *hcl.EvalContext, block *hcl.Block) (*Matrix, hcl.Diagnostics) {
	mContent, diags := block.Body.Content(scenarioSchema)
	if diags.HasErrors() {
		return nil, diags
	}

	mBlocks := mContent.Blocks.OfType(blockTypeMatrix)
	switch len(mBlocks) {
	case 0:
		// We have no matrix block defined
		return nil, diags
	case 1:
		// Continue
		break
	default:
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "scenarios has more than one matrix block defined",
			Detail:   fmt.Sprintf("up to one matrix block is expected, found %d", len(mBlocks)),
			Subject:  block.TypeRange.Ptr(),
			Context:  block.DefRange.Ptr(),
		})
	}

	// Let's decode our matrix block into a matrix
	block = mBlocks[0]
	matrix := NewMatrix()

	decodeMatrixAttribute := func(block *hcl.Block, attr *hcl.Attribute) (Vector, hcl.Diagnostics) {
		var diags hcl.Diagnostics
		vec := Vector{}

		val, moreDiags := attr.Expr.Value(ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			return vec, diags
		}

		if !val.CanIterateElements() {
			return vec, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "matrix attribute value must be a list of strings",
				Detail:   fmt.Sprintf("expected value for %s to be a list of strings, found %s", attr.Name, val.Type().GoString()),
				Subject:  attr.NameRange.Ptr(),
				Context:  block.DefRange.Ptr(),
			})
		}

		if len(val.AsValueSlice()) == 0 {
			return vec, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "matrix attribute values cannot be empty lists",
				Subject:  attr.NameRange.Ptr(),
				Context:  block.DefRange.Ptr(),
			})
		}

		for _, elm := range val.AsValueSlice() {
			if !elm.Type().Equals(cty.String) {
				return vec, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "matrix attribute value must be a list of strings",
					Detail:   fmt.Sprintf("found element with type %s", elm.GoString()),
					Subject:  attr.NameRange.Ptr(),
					Context:  block.DefRange.Ptr(),
				})
			}

			vec = append(vec, NewElement(attr.Name, elm.AsString()))
		}

		return vec, diags
	}

	// Go maps are intentionally unordered. We need to sort our attributes
	// so that our variants elements are deterministic every time we
	// decode our flightplan.
	sortAttributes := func(attrs map[string]*hcl.Attribute) []*hcl.Attribute {
		sorted := []*hcl.Attribute{}
		for _, attr := range attrs {
			sorted = append(sorted, attr)
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Name < sorted[j].Name
		})

		return sorted
	}

	// Each attribute in the matrix should be a variant name whose value must
	// be a list of strings. Convert the value into a matrix vector and add it.
	// We're ignoring the diagnostics JustAttributes() will return here because
	// there might also be include and exclude blocks.
	mAttrs, _ := block.Body.JustAttributes()
	for _, attr := range sortAttributes(mAttrs) {
		vec, moreDiags := decodeMatrixAttribute(block, attr)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		matrix.AddVector(vec)
	}

	// Now that we have our basic variant vectors in our matrix, we need to combine
	// all vectors into a product that matches all possible unique value combinations.
	matrix = matrix.CartesianProduct().UniqueValues()

	// Now we need to go through all of our blocks and process include and exclude
	// directives. Since HCL allows us to use ordering we'll apply them in the
	// order in which they're defined.
	blockC, remain, moreDiags := block.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: blockTypeMatrixInclude},
			{Type: blockTypeMatrixExclude},
		},
	})
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}
	diags = diags.Extend(verifyBodyOnlyHasBlocksWithLabels(
		remain, blockTypeMatrixInclude, blockTypeMatrixExclude,
	))

	for _, mBlock := range blockC.Blocks {
		switch mBlock.Type {
		case "include":
			iMatrix := NewMatrix()
			iAttrs, moreDiags := mBlock.Body.JustAttributes()
			diags = diags.Extend(moreDiags)
			if moreDiags.HasErrors() {
				continue
			}

			for _, attr := range sortAttributes(iAttrs) {
				vec, moreDiags := decodeMatrixAttribute(mBlock, attr)
				diags = diags.Extend(moreDiags)
				if moreDiags.HasErrors() {
					continue
				}

				iMatrix.AddVector(vec)
			}

			// Generate our possible include vectors and add them to our main
			// matrix.
			for _, vec := range iMatrix.CartesianProduct().UniqueValues().Vectors {
				matrix.AddVector(vec)
			}
		case "exclude":
			eMatrix := NewMatrix()
			eAttrs, moreDiags := mBlock.Body.JustAttributes()
			diags = diags.Extend(moreDiags)
			if moreDiags.HasErrors() {
				continue
			}

			for _, attr := range sortAttributes(eAttrs) {
				vec, moreDiags := decodeMatrixAttribute(mBlock, attr)
				diags = diags.Extend(moreDiags)
				if moreDiags.HasErrors() {
					continue
				}
				eMatrix.AddVector(vec)
			}

			excludes := []*Exclude{}
			for _, vec := range eMatrix.CartesianProduct().UniqueValues().Vectors {
				ex, err := NewExclude(pb.Scenario_Filter_Exclude_MODE_CONTAINS, vec)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "unable to generate exclusion filter",
						Detail:   err.Error(),
						Subject:  hcl.RangeBetween(mBlock.LabelRanges[0], mBlock.LabelRanges[1]).Ptr(),
						Context:  mBlock.DefRange.Ptr(),
					})
				}
				excludes = append(excludes, ex)
			}

			// Update our matrix to a copy which has vectors which match our exclusions
			matrix = matrix.Exclude(excludes...)
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "invalid block in matrix",
				Detail:   fmt.Sprintf("blocks of type include and exclude are supported in matrix blocks, found %s", mBlock.Type),
				Subject:  mBlock.TypeRange.Ptr(),
				Context:  mBlock.DefRange.Ptr(),
			})
			continue
		}
	}

	// Return our matrix but do one final pass removing any duplicates that might
	// have been introduced during our inclusions.
	return matrix.UniqueValues(), diags
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
	var diags hcl.Diagnostics
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
	var diags hcl.Diagnostics

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
	var diags hcl.Diagnostics

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
	var diags hcl.Diagnostics

	if len(block.Labels) == 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid scenario block",
			Detail:   "scenario blocks can only have a single name label",
			Subject:  block.TypeRange.Ptr(),
			Context:  hcl.RangeBetween(block.TypeRange, block.DefRange).Ptr(),
		})

		return diags
	}

	for i, label := range block.Labels {
		// Make sure it's a valid HCL identifier
		if !hclsyntax.ValidIdentifier(label) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "block label is invalid",
				Detail:   "block label is not a valid HCL identifier",
				Subject:  block.LabelRanges[i].Ptr(),
			})
		}

		// Make sure it also adheres to Enos block name rules
		r := regexp.MustCompile(`^[\w]+$`)
		if !r.Match([]byte(label)) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "block label is invalid",
				Detail:   "block label is not a valid enos identifier",
				Subject:  block.LabelRanges[i].Ptr(),
			})
		}
	}

	return diags
}

// verifyBlockHasNLabels verifies that the given block has the appropriate number
// of defined labels.
func verifyBlockHasNLabels(block *hcl.Block, n int) hcl.Diagnostics {
	var diags hcl.Diagnostics

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
