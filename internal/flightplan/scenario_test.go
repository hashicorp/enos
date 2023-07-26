package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test_Decode_Scenario tests decoding a scenario.
func Test_Decode_Scenario(t *testing.T) {
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
			desc: "invalid enos identifier scenario block",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "hascolon:" {
  step "first" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "invalid block in scenario",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  notablock "something" {
    something = "else"
  }

  step "first" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "invalid attr in scenario",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  notanattr = "foo"

  step "first" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "locals",
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  locals {
    something = "another"
    another   = local.something
    mod       = module.backend.name
  }

  step "first" {
    module = local.mod
  }

  output "another" {
    value = local.another
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
						Name:         "backend",
						TerraformCLI: DefaultTerraformCLI(),
						Steps: []*ScenarioStep{
							{
								Name: "first",
								Module: &Module{
									Name:   "backend",
									Source: modulePath,
									Attrs:  map[string]cty.Value{},
								},
							},
						},
						Outputs: []*ScenarioOutput{
							{
								Name:  "another",
								Value: testMakeStepVarValue(cty.StringVal("another")),
							},
						},
					},
				},
			},
		},
	} {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			fp, err := testDecodeHCL(t, []byte(test.hcl))
			if test.fail {
				require.Error(t, err)

				return
			}
			require.NoError(t, err)
			testRequireEqualFP(t, fp, test.expected)
		})
	}
}
