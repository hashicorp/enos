// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test_Module_EvalContext_Functions tests a few built-in functions to ensure
// that they're available when the module blocks are evaluated.
func Test_Module_EvalContext_Functions(t *testing.T) {
	t.Parallel()

	modulePath, err := filepath.Abs("./tests/simple_module")
	require.NoError(t, err)

	// This isn't an exhaustive test of all functions, but we should have
	// access to functions in the base resource context.
	for _, test := range []struct {
		desc     string
		expr     string
		expected string
	}{
		{
			desc:     "upper",
			expr:     `upper("low")`,
			expected: "LOW",
		},
		{
			desc:     "trimsuffix",
			expr:     `trimsuffix("something.com", ".com")`,
			expected: "something",
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			hcl := fmt.Sprintf(`
module "backend" {
  source = "%s"
  something = %s
}

scenario "basic" {
  step "first" {
    module = module.backend
  }
}`, modulePath, test.expr)
			fp, err := testDecodeHCL(t, []byte(hcl), DecodeTargetAll)
			require.NoError(t, err)
			require.Len(t, fp.Modules, 1)
			v, ok := fp.Modules[0].Attrs["something"]
			require.True(t, ok)
			require.Equal(t, test.expected, v.AsString())
		})
	}
}

// Test_Decode_Module tests module decoding.
func Test_Decode_Module(t *testing.T) {
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
			desc: "source registry with version",
			hcl: `
module "backend" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.11.0"
}

scenario "basic" {
  step "first" {
    module = module.backend
  }
}
`,
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				Modules: []*Module{
					{
						Name:    "backend",
						Source:  "terraform-aws-modules/vpc/aws",
						Version: "3.11.0",
					},
				},
				ScenarioBlocks: ScenarioBlocks{
					{
						Name: "basic",
						Scenarios: []*Scenario{
							{
								Name:         "basic",
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name: "first",
										Module: &Module{
											Name:    "backend",
											Source:  "terraform-aws-modules/vpc/aws",
											Version: "3.11.0",
											Attrs:   map[string]cty.Value{},
										},
									},
								},
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
module "hascolon:" {
  source = "%s"
}

scenario "basic" {
  step "first" {
    module = module.hascolon
  }
}
`, modulePath),
		},
		{
			desc: "count meta-arg attr",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
  count = 1
}

scenario "backend" {
  step "first" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "for_each meta-arg attr",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
  for_each = toset(["1", "2"])
}

scenario "backend" {
  step "first" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "depends_on meta-arg attr",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

module "frontend" {
  source = "%[1]s"
  depends_on = module.backend
}

scenario "backend" {
  step "first" {
    module = module.backend
  }
}
`, modulePath),
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			fp, err := testDecodeHCL(t, []byte(test.hcl), DecodeTargetAll)
			if test.fail {
				require.Error(t, err)

				return
			}
			require.NoError(t, err)
			testRequireEqualFP(t, fp, test.expected)
		})
	}
}
