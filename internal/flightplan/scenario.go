package flightplan

import (
	"fmt"

	hcl "github.com/hashicorp/hcl/v2"
)

var scenarioSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: blockTypeScenarioStep, LabelNames: []string{"name"}},
	},
}

// Scenario represents an Enos scenario
type Scenario struct {
	Name  string
	Steps []*ScenarioStep
}

// NewScenario returns a new Scenario
func NewScenario() *Scenario {
	return &Scenario{
		Steps: []*ScenarioStep{},
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
	foundSteps := 0
	for _, childBlock := range content.Blocks {
		switch childBlock.Type {
		case blockTypeScenarioStep:
			foundSteps++
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

			s.Steps = append(s.Steps, step)
		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "unknown block in scenario",
				Detail:   fmt.Sprintf(`unable to parse unknown block "%s" in scenario`, childBlock.Type),
				Subject:  &childBlock.DefRange,
			})
		}
	}

	if foundSteps == 0 {
		r := block.Body.MissingItemRange()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "missing required step block",
			Detail:   "scenarios require one or more step blocks",
			Subject:  &r,
		})
	}

	return diags
}
