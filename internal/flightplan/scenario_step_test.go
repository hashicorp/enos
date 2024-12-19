// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test_Decode_Scenario_Step tests decoding of scenario steps.
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
				ScenarioBlocks: ScenarioBlocks{
					{
						Name: "basic",
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
			desc: "step skip invalid value",
			fail: true,
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

scenario "skipper" {
  step "one" {
    skip_step = "mayonaise"
    module = module.one
  }
}
`, modulePath),
		},
		{
			desc: "step skip valid",
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

scenario "skipper" {
  step "one" {
    skip_step = true
    module = module.one
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
					},
				},
				ScenarioBlocks: ScenarioBlocks{
					{
						Name: "skipper",
						Scenarios: []*Scenario{
							{
								Name:         "skipper",
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name:   "one",
										Skip:   true,
										Module: NewModule(),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "step depends_on invalid string",
			fail: true,
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

scenario "depends_on" {
  step "one" {
    module = module.one
  }

  step "two" {
    depends_on = ["nope"]
    module = module.one
  }
}
`, modulePath),
		},
		{
			desc: "step depends_on valid string twice",
			fail: true,
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

scenario "depends_on" {
  step "one" {
    module = module.one
  }

  step "two" {
    depends_on = ["one", "one"]
    module = module.one
  }
}
`, modulePath),
		},
		{
			desc: "step depends_on invalid step ref",
			fail: true,
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

scenario "depends_on" {
  step "one" {
    module = module.one
  }

  step "two" {
    depends_on = [step.nope]
    module = module.one
  }
}
`, modulePath),
		},
		{
			desc: "step depends_on valid ref twice",
			fail: true,
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

scenario "depends_on" {
  step "one" {
    module = module.one
  }

  step "two" {
    depends_on = [step.one, step.one]
    module = module.one
  }
}
`, modulePath),
		},
		{
			desc: "step depends_on skipped step",
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

scenario "depends_on_skipped" {
  step "one" {
	skip_step = true
    module = module.one
  }

  step "two" {
    module = module.one
  }

  step "three" {
    depends_on = [step.one, step.two]
    module = module.one
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
					},
				},
				ScenarioBlocks: ScenarioBlocks{
					{
						Name: "depends_on_skipped",
						Scenarios: []*Scenario{
							{
								Name:         "depends_on_skipped",
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name:   "one",
										Skip:   true,
										Module: NewModule(),
									},
									{
										Name:   "two",
										Module: &Module{Name: "one", Source: modulePath},
									},
									{
										Name:      "three",
										DependsOn: []string{"two"},
										Module:    &Module{Name: "one", Source: modulePath},
									},
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "step depends_on valid string and ref mixed",
			fail: true,
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

scenario "depends_on" {
  step "one" {
    module = module.one
  }

  step "two" {
    depends_on = ["one", step.one]
    module = module.one
  }
}
`, modulePath),
		},
		{
			desc: "step description and verifies",
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

quality "tests_pass" {
  description = "the tests pass"
}

quality "data_is_durable" {
  description = "the data is durable"
}

scenario "test_verifies" {
  step "verifies_singular" {
    description = "test a single verifies"

    verifies = quality.tests_pass

    module = module.one
  }

  step "verifies_mutiple_and_inline" {
    description = "tests multiple and inline verifies"

    verifies = [
			{ name: "inline", description: "inline quality" },
      quality.data_is_durable,
    ]

    module = module.one
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
					},
				},
				Qualities: []*Quality{
					{
						Name:        "tests_pass",
						Description: "the tests pass",
					},
					{
						Name:        "data_is_durable",
						Description: "the data is durable",
					},
				},
				ScenarioBlocks: ScenarioBlocks{
					{
						Name: "test_verifies",
						Scenarios: []*Scenario{
							{
								Name:         "test_verifies",
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name:        "verifies_singular",
										Description: "test a single verifies",
										Verifies: []*Quality{
											{
												Name:        "tests_pass",
												Description: "the tests pass",
											},
										},
										Module: &Module{
											Name:   "one",
											Source: modulePath,
										},
									},
									{
										Name:        "verifies_mutiple_and_inline",
										Description: "tests multiple and inline verifies",
										Verifies: []*Quality{
											{
												Name:        "data_is_durable",
												Description: "the data is durable",
											},
											{
												Name:        "inline",
												Description: "inline quality",
											},
										},
										Module: &Module{
											Name:   "one",
											Source: modulePath,
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
			desc: "step depends_on valid string",
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

scenario "depends_on" {
  step "one" {
    module = module.one
  }

  step "two" {
    depends_on = ["one"]
    module = module.one
  }

  step "three" {
    depends_on = ["one", "two"]
    module = module.one
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
					},
				},
				ScenarioBlocks: ScenarioBlocks{
					{
						Name: "depends_on",
						Scenarios: []*Scenario{
							{
								Name:         "depends_on",
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name: "one",
										Module: &Module{
											Name:   "one",
											Source: modulePath,
										},
									},
									{
										Name: "two",
										Module: &Module{
											Name:   "one",
											Source: modulePath,
										},
										DependsOn: []string{"one"},
									},
									{
										Name: "three",
										Module: &Module{
											Name:   "one",
											Source: modulePath,
										},
										DependsOn: []string{"one", "two"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "step depends_on valid ref",
			hcl: fmt.Sprintf(`
module "one" {
  source = "%s"
}

scenario "depends_on" {
  step "one" {
    module = module.one
  }

  step "two" {
    depends_on = [step.one]
    module = module.one
  }

  step "three" {
    depends_on = [step.one, step.two]
    module = module.one
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
					},
				},
				ScenarioBlocks: ScenarioBlocks{
					{
						Name: "depends_on",
						Scenarios: []*Scenario{
							{
								Name:         "depends_on",
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name: "one",
										Module: &Module{
											Name:   "one",
											Source: modulePath,
										},
									},
									{
										Name: "two",
										Module: &Module{
											Name:   "one",
											Source: modulePath,
										},
										DependsOn: []string{"one"},
									},
									{
										Name: "three",
										Module: &Module{
											Name:   "one",
											Source: modulePath,
										},
										DependsOn: []string{"one", "two"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "step variables",
			hcl: fmt.Sprintf(`
variable "something" {
  default = "somethingval"
  type = string
}

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
      concrete            = "twoconcrete"
      reference           = step.one.reference
      reference_same_name = step.one.concrete
      inherited_concrete  = step.one.variables.concrete
      oneattr             = step.one.oneattr
      matrixconcrete      = matrix.input
      matrixreference     = step.one.matrixinput
      fromvariables       = var.something
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
				ScenarioBlocks: ScenarioBlocks{
					{
						Name: "step_vars",
						Scenarios: []*Scenario{
							{
								Name:         "step_vars",
								Variants:     NewVector(NewElement("input", "matrixinput")),
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
												"twoattr":             testMakeStepVarValue(cty.StringVal("twoattrval")),
												"concrete":            testMakeStepVarValue(cty.StringVal("twoconcrete")),
												"inherited_concrete":  testMakeStepVarValue(cty.StringVal("oneconcrete")),
												"reference":           testMakeStepVarTraversal("step", "one", "reference"),
												"reference_same_name": testMakeStepVarTraversal("step", "one", "concrete"),
												"oneattr":             testMakeStepVarTraversal("step", "one", "oneattr"),
												"matrixconcrete":      testMakeStepVarValue(cty.StringVal("matrixinput")),
												"matrixreference":     testMakeStepVarTraversal("step", "one", "matrixinput"),
												"fromvariables":       testMakeStepVarValue(cty.StringVal("somethingval")),
											},
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
			desc: "step variables with vars and outputs of same name",
			hcl: fmt.Sprintf(`
module "cluster" {
  source = "%s"
}

module "worker" {
  source = "%[1]s"
}

variable "addr" {
  type = string
  default = "http://192.168.0.1"
}

scenario "boundary" {
  step "cluster" {
    module = module.cluster

    variables {
      addr = var.addr
    }
  }

  step "worker" {
    module = module.worker

    variables {
      upstream_addr = step.cluster.addr
    }
  }

  step "worker_downstream" {
    module = module.worker

    variables {
      upstream_addr = step.worker.upstream_addr
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
						Name:   "cluster",
						Source: modulePath,
					},
					{
						Name:   "worker",
						Source: modulePath,
					},
				},
				ScenarioBlocks: ScenarioBlocks{
					{
						Name: "boundary",
						Scenarios: []*Scenario{
							{
								Name:         "boundary",
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name: "cluster",
										Module: &Module{
											Name:   "cluster",
											Source: modulePath,
											Attrs: map[string]cty.Value{
												"addr": testMakeStepVarValue(cty.StringVal("http://192.168.0.1")),
											},
										},
									},
									{
										Name: "worker",
										Module: &Module{
											Name:   "worker",
											Source: modulePath,
											Attrs: map[string]cty.Value{
												"upstream_addr": testMakeStepVarTraversal("step", "cluster", "addr"),
											},
										},
									},
									{
										Name: "worker_downstream",
										Module: &Module{
											Name:   "worker",
											Source: modulePath,
											Attrs: map[string]cty.Value{
												"upstream_addr": testMakeStepVarTraversal("step", "worker", "upstream_addr"),
											},
										},
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
