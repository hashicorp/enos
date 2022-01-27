package flightplan

import (
	"fmt"

	"github.com/zclconf/go-cty/cty/gocty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var scenarioSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "terraform_cli", Required: false},
		{Name: "transport", Required: false},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: blockTypeScenarioStep, LabelNames: []string{"name"}},
	},
}

// Scenario represents an Enos scenario
type Scenario struct {
	Name         string
	TerraformCLI *TerraformCLI
	Transport    *Transport
	Steps        []*ScenarioStep
}

// NewScenario returns a new Scenario
func NewScenario() *Scenario {
	return &Scenario{
		TerraformCLI: NewTerraformCLI(),
		Steps:        []*ScenarioStep{},
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

	// Decode the scenario transport reference
	moreDiags = s.decodeAndValidateTransportAttribute(content, ctx)
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

// decodeAndValidateTransportAttribute decodess the transport attribute
// from the content and validates that it refers to an existing transport.
func (s *Scenario) decodeAndValidateTransportAttribute(
	content *hcl.BodyContent,
	ctx *hcl.EvalContext,
) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// See if we've set an transport
	enosTransport, ok := content.Attributes["transport"]
	if ok {
		s.Transport = NewTransport()
		// Decode our transport from the eval context. If it hasn't been defined
		// this will raise an error.
		moreDiags := gohcl.DecodeExpression(enosTransport.Expr, ctx, &s.Transport)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			return diags
		}

		return diags
	}

	// We haven't been configured to use an transport, so lets set
	// it to the default if it exists
	enosTransports, err := findEvalContextVariable("transport", ctx)
	if err != nil {
		// We don't have an transport's in the eval context so we
		// get to move on.
		return diags
	}

	// Find default and set it one exists
	defaultTransport, ok := enosTransports.AsValueMap()["default"]
	if !ok {
		return diags
	}

	s.Transport = NewTransport()
	err = gocty.FromCtyValue(defaultTransport, &s.Transport)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to convert default transport from eval context to object",
			Detail:   err.Error(),
			Subject:  content.MissingItemRange.Ptr(),
		})
	}

	return diags
}
