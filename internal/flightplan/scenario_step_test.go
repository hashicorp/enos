package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test_Decode_Scenario_Step tests decoding of scenario steps
func Test_Decode_Scenario_Step(t *testing.T) {
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
			desc: "invalid module reference",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    module = module.not_real
  }
}
`, modulePath),
		},
		{
			desc: "valid module reference",
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"

  driver = "postgres"
}

module "frontend_blue" {
  source = "%[1]s"

  app_version = "1.0.0"
}

module "frontend_green" {
  source = "%[1]s"

  app_version = "1.1.0"
}

module "frontend_red" {
  source = "hashicorp/qti/frontend-aws"

  version = "2.0.0"
}

scenario "basic" {
  step "backend" {
    module = module.backend
  }

  step "frontend_blue" {
    module = "frontend_blue"
  }

  step "frontend_green" {
    module = module.frontend_green
  }

  step "frontend_red" {
    module = "frontend_red"
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
						Attrs: map[string]cty.Value{
							"driver": cty.StringVal("postgres"),
						},
					},
					{
						Name:   "frontend_blue",
						Source: modulePath,
						Attrs: map[string]cty.Value{
							"app_version": cty.StringVal("1.0.0"),
						},
					},
					{
						Name:   "frontend_green",
						Source: modulePath,
						Attrs: map[string]cty.Value{
							"app_version": cty.StringVal("1.1.0"),
						},
					},
					{
						Name:    "frontend_red",
						Version: "2.0.0",
						Source:  "hashicorp/qti/frontend-aws",
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
									Attrs: map[string]cty.Value{
										"driver": testMakeStepVarValue(cty.StringVal("postgres")),
									},
								},
							},
							{
								Name: "frontend_blue",
								Module: &Module{
									Name:   "frontend_blue",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"app_version": testMakeStepVarValue(cty.StringVal("1.0.0")),
									},
								},
							},
							{
								Name: "frontend_green",
								Module: &Module{
									Name:   "frontend_green",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"app_version": testMakeStepVarValue(cty.StringVal("1.1.0")),
									},
								},
							},
							{
								Name: "frontend_red",
								Module: &Module{
									Name:    "frontend_red",
									Source:  "hashicorp/qti/frontend-aws",
									Version: "2.0.0",
									Attrs:   map[string]cty.Value{},
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
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "hascolon:" {
    module = module.backend
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
    notablock "something" {
      something = "else"
    }

    module = module.backend
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
    notanattr = "foo"
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "count meta-arg attr in variables",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    module = module.backend
    variables {
      count = 1
    }
  }
}
`, modulePath),
		},
		{
			desc: "for_each meta-arg attr in variables",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    variables {
      for_each = toset(["1", "2"])
    }
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "depends_on meta-arg attr in variables",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

module "frontend" {
  source = "%[1]s"
}

scenario "backend" {
  step "first" {
    variables {
      depends_on = module.backend
    }
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "redeclared step",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    module = module.backend
  }

  step "first" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "step variables",
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"

  oneattr = "oneattrval"
}

module "two" {
  source = "%[1]s"

  twoattr = "twoattrval"
}

scenario "step_vars" {
  matrix {
    input = ["matrixinput"]
  }

  step "one" {
    module = module.one

	variables {
      concrete    = "oneconcrete"
	  matrixinput = matrix.input
	}
  }

  step "two" {
    module = module.two

	variables {
      concrete           = "twoconcrete"
	  inherited_concrete = step.one.concrete
	  reference          = step.one.reference
	  oneattr            = step.one.oneattr
	  matrixconcrete     = matrix.input
	  matrixinherited    = step.one.matrixinput
	}
  }
}
`, modulePath),
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				Modules: []*Module{
					{
						Name:   "one",
						Source: modulePath,
						Attrs: map[string]cty.Value{
							"oneattr": cty.StringVal("oneattrval"),
						},
					},
					{
						Name:   "two",
						Source: modulePath,
						Attrs: map[string]cty.Value{
							"twoattr": cty.StringVal("twoattrval"),
						},
					},
				},
				Scenarios: []*Scenario{
					{
						Name:         "step_vars",
						Variants:     Vector{Element{Key: "input", Val: "matrixinput"}},
						TerraformCLI: DefaultTerraformCLI(),
						Steps: []*ScenarioStep{
							{
								Name: "one",
								Module: &Module{
									Name:   "one",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"oneattr":     testMakeStepVarValue(cty.StringVal("oneattrval")),
										"concrete":    testMakeStepVarValue(cty.StringVal("oneconcrete")),
										"matrixinput": testMakeStepVarValue(cty.StringVal("matrixinput")),
									},
								},
							},
							{
								Name: "two",
								Module: &Module{
									Name:   "two",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"twoattr":            testMakeStepVarValue(cty.StringVal("twoattrval")),
										"concrete":           testMakeStepVarValue(cty.StringVal("twoconcrete")),
										"inherited_concrete": testMakeStepVarValue(cty.StringVal("oneconcrete")),
										"reference":          testMakeStepVarTraversal("step", "one", "reference"),
										"oneattr":            testMakeStepVarValue(cty.StringVal("oneattrval")),
										"matrixconcrete":     testMakeStepVarValue(cty.StringVal("matrixinput")),
										"matrixinherited":    testMakeStepVarValue(cty.StringVal("matrixinput")),
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
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
