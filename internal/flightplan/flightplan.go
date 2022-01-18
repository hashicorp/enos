package flightplan

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

const (
	blockTypeTerraformCLI = "terraform_cli"
	blockTypeTransport    = "transport"
	blockTypeModule       = "module"
	blockTypeScenario     = "scenario"
	blockTypeScenarioStep = "step"
)

var flightPlanSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: blockTypeTransport, LabelNames: []string{"name"}},
		{Type: blockTypeTerraformCLI, LabelNames: []string{"name"}},
		{Type: blockTypeScenario, LabelNames: []string{"name"}},
		{Type: blockTypeModule, LabelNames: []string{"name"}},
	},
}

// NewFlightPlan returns a new instance of a FlightPlan
func NewFlightPlan(opts ...Opt) (*FlightPlan, error) {
	fp := &FlightPlan{
		Files:         map[string]*hcl.File{},
		TerraformCLIs: []*TerraformCLI{},
		Transports:    []*Transport{},
		Scenarios:     []*Scenario{},
		Modules:       []*Module{},
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
	BaseDir       string
	BodyContent   *hcl.BodyContent
	Files         map[string]*hcl.File
	TerraformCLIs []*TerraformCLI
	Transports    []*Transport
	Scenarios     []*Scenario
	Modules       []*Module
}

// Decode takes a base eval content and HCL body and decodes it in chunks,
// continually expanding the evaluation context as more sub-sections are
// decoded. It returns HCL diagnostics that are collected over the course of
// decoding.
func (fp *FlightPlan) Decode(ctx *hcl.EvalContext, body hcl.Body, files map[string]*hcl.File) hcl.Diagnostics {
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

	fp.Files = files

	// Decode our top-level schema
	fp.BodyContent, diags = body.Content(flightPlanSchema)

	// decode sub-sections. Each sub-section decoder is responsible for
	// extending the evaluation context for further evaluation.
	diags = diags.Extend(fp.decodeTransports(ctx))
	diags = diags.Extend(fp.decodeTerraformCLIs(ctx))
	diags = diags.Extend(fp.decodeModules(ctx))
	diags = diags.Extend(fp.decodeScenarios(ctx))

	return diags
}

// decodeTransports decodes "transport" blocks that are defined in the
// top-level schema.
func (fp *FlightPlan) decodeTransports(ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics
	transports := map[string]cty.Value{}

	for _, block := range fp.BodyContent.Blocks.OfType(blockTypeTransport) {
		moreDiags := verifyBlockLabelsAreValidIdentifiers(block)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		transport := NewTransport()
		moreDiags = transport.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		fp.Transports = append(fp.Transports, transport)

		var err error
		transports[transport.Name], err = transport.evalCtx()
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unable to generate eval context for transport",
				Detail:   err.Error(),
				Subject:  block.TypeRange.Ptr(),
				Context:  block.DefRange.Ptr(),
			})
		}
	}

	ctx.Variables["transport"] = cty.ObjectVal(transports)

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
		mods[module.Name] = module.evalCtx()
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

		scenario := NewScenario()
		moreDiags = scenario.decode(block, ctx.NewChild())
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		fp.Scenarios = append(fp.Scenarios, scenario)
	}

	// NOTE: when we add variants we'll need to also sort by variants.
	sort.Slice(fp.Scenarios, func(i, j int) bool {
		return fp.Scenarios[i].Name < fp.Scenarios[j].Name
	})

	return diags
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
				Summary:  "invalid module attribute",
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
				Summary:  "invalid block",
				Detail:   "sub-blocks are not allowed",
				Subject:  block.TypeRange.Ptr(),
				Context:  hcl.RangeBetween(block.TypeRange, block.Range()).Ptr(),
			})
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
