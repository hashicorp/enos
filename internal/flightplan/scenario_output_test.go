package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test_Decode_Scenario_Output tests decoding of scenario outputs.
func Test_Decode_Scenario_Output(t *testing.T) {
	t.Parallel()

	modulePath, err := filepath.Abs("./tests/simple_module")
	require.NoError(t, err)

	for _, test := range []struct {
		desc     string
		hcl      string
		expected *FlightPlan
		fail     bool
	}{
		{
			desc: "valid outputs",
			hcl: fmt.Sprintf(`
variable "input" {
  type    = string
  default = "defaultval"
}

module "backend" {
  source = "%s"
}

scenario "basic" {
  step "backend" {
    module = module.backend
  }

  output "static" {
    description = "static output"
    sensitive   = true
    value       = "veryknown"
  }

  output "var" {
    description = "from variable"
    value = var.input
  }

  output "step_ref" {
    description = "step module output"
    value       = step.backend.addrs
  }
}
`, modulePath),
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				Modules: []*Module{
					{
						Name:   "backend",
						Source: modulePath,
						Attrs:  map[string]cty.Value{},
					},
				},
				Scenarios: []*Scenario{
					{
						Name:         "basic",
						TerraformCLI: DefaultTerraformCLI(),
						Steps: []*ScenarioStep{
							{
								Name: "backend",
								Module: &Module{
									Name:   "backend",
									Source: modulePath,
									Attrs:  map[string]cty.Value{},
								},
							},
						},
						Outputs: []*ScenarioOutput{
							{
								Name:        "static",
								Description: "static output",
								Sensitive:   true,
								Value:       testMakeStepVarValue(cty.StringVal("veryknown")),
							},
							{
								Name:        "var",
								Description: "from variable",
								Value:       testMakeStepVarValue(cty.StringVal("fromenv")),
							},
							{
								Name:        "step_ref",
								Description: "step module output",
								Value:       testMakeStepVarTraversal("step", "backend", "addrs"),
							},
						},
					},
				},
			},
		},
		{
			desc: "invalid identifier",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "backend" {
    module = module.backend
  }

  output ":hascolon" {
    value = "foo"
  }
}
`, modulePath),
		},
		{
			desc: "invalid block",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    module = module.backend

    output "something" {
      notablock "something" {
        something = "else"
      }
    }
  }
}
`, modulePath),
		},
		{
			desc: "invalid attr",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    module = module.backend

    output "something" {
      notanattr = "something"
      value     = "foo"
    }
  }
}
`, modulePath),
		},
		{
			desc: "redeclared output",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    module = module.backend
  }

  output "static" {
    description = "static output"
    value       = "veryknown"
  }

  output "static" {
    description = "static output"
    value       = "veryknown"
  }
}
`, modulePath),
		},
	} {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			fp, err := testDecodeHCL(t, []byte(test.hcl), "ENOS_VAR_input=fromenv")
			if test.fail {
				require.Error(t, err)

				return
			}
			require.NoError(t, err)
			testRequireEqualFP(t, fp, test.expected)
		})
	}
}
