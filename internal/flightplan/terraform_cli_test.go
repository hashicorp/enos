package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test_Decode_TerraformCLI
func Test_Decode_TerraformCLI(t *testing.T) {
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
			desc: "invalid identifier",
			fail: true,
			hcl: fmt.Sprintf(`
terraform_cli ":hascolon" {
}

module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "invalid block",
			fail: true,
			hcl: fmt.Sprintf(`
terraform_cli "debug" {
  not_a_block "foo" {
    foo = "bar"
  }
}

module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "invalid attr",
			fail: true,
			hcl: fmt.Sprintf(`
terraform_cli "debug" {
  not_an_attr = "something"
}

module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "invalid reference",
			fail: true,
			hcl: fmt.Sprintf(`
terraform_cli "debug" {
  path = "/opt/usr/bin/terraform"

  env = {
    TF_LOG_CORE     = "off"
    TF_LOG_PROVIDER = "debug"
  }
}

module "backend" {
  source = "%s"

  driver = "postgres"
}

scenario "default" {
  step "backend" {
    module = module.backend
  }
}

scenario "debug" {
  terraform_cli = terraform_cli.bad_ref

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "maximal configuration",
			hcl: fmt.Sprintf(`
terraform_cli "debug" {
  path                         = "/opt/usr/bin/terraform"
  disable_checkpoint           = true
  disable_checkpoint_signature = true
  plugin_cache_dir             = "$HOME/.terraform.d/plugin-cache"

  env = {
    TF_LOG_CORE     = "off"
    TF_LOG_PROVIDER = "debug"
  }

  credentials "app.terraform.io" {
    token = "supersecret"
  }

  credentials_helper "credstore" {
    args = ["--host=credstore.example.com"]
  }

  provider_installation {
    dev_overrides = {
      "hashicorp/null" = "/home/developer/tmp/terraform-null"
    }

    filesystem_mirror {
      path    = "/usr/share/terraform/providers"
      include = ["examplef.com/*/*"]
    }

   network_mirror {
      url    = "https://providers.example.com"
      include = ["examplen.com/*/*"]
    }

    direct {
      exclude = ["exampled.com/*/*"]
    }
  }
}

module "backend" {
  source = "%s"

  driver = "postgres"
}

scenario "default" {
  step "backend" {
    module = module.backend
  }
}

scenario "debug" {
  terraform_cli = terraform_cli.debug

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					{
						Name: "debug",
						Path: "/opt/usr/bin/terraform",
						Env: map[string]string{
							"TF_LOG_CORE":     "off",
							"TF_LOG_PROVIDER": "debug",
						},
						ConfigVal: cty.ObjectVal(map[string]cty.Value{
							"disable_checkpoint":           cty.BoolVal(true),
							"disable_checkpoint_signature": cty.BoolVal(true),
							"plugin_cache_dir":             cty.StringVal("$HOME/.terraform.d/plugin-cache"),
							"credentials": cty.MapVal(map[string]cty.Value{
								"app.terraform.io": cty.ObjectVal(map[string]cty.Value{
									"token": cty.StringVal("supersecret"),
								}),
							}),
							"credentials_helper": cty.MapVal(map[string]cty.Value{
								"credstore": cty.ObjectVal(map[string]cty.Value{
									"args": cty.ListVal([]cty.Value{cty.StringVal("--host=credstore.example.com")}),
								}),
							}),
							"provider_installation": cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									"dev_overrides": cty.MapVal(map[string]cty.Value{
										"hashicorp/null": cty.StringVal("/home/developer/tmp/terraform-null"),
									}),
									"filesystem_mirror": cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											"path": cty.StringVal("/usr/share/terraform/providers"),
											"include": cty.ListVal([]cty.Value{
												cty.StringVal("examplef.com/*/*"),
											}),
											"exclude": cty.NullVal(cty.List(cty.String)),
										}),
									}),
									"network_mirror": cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											"url": cty.StringVal("https://providers.example.com"),
											"include": cty.ListVal([]cty.Value{
												cty.StringVal("examplen.com/*/*"),
											}),
											"exclude": cty.NullVal(cty.List(cty.String)),
										}),
									}),
									"direct": cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											"exclude": cty.ListVal([]cty.Value{
												cty.StringVal("exampled.com/*/*"),
											}),
											"include": cty.NullVal(cty.List(cty.String)),
										}),
									}),
								}),
							}),
						}),
					},
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
				},
				Scenarios: []*Scenario{
					{
						Name: "debug",
						TerraformCLI: &TerraformCLI{
							Name: "debug",
							Path: "/opt/usr/bin/terraform",
							Env: map[string]string{
								"TF_LOG_CORE":     "off",
								"TF_LOG_PROVIDER": "debug",
							},
							ConfigVal: cty.ObjectVal(map[string]cty.Value{
								"disable_checkpoint":           cty.BoolVal(true),
								"disable_checkpoint_signature": cty.BoolVal(true),
								"plugin_cache_dir":             cty.StringVal("$HOME/.terraform.d/plugin-cache"),
								"credentials": cty.MapVal(map[string]cty.Value{
									"app.terraform.io": cty.ObjectVal(map[string]cty.Value{
										"token": cty.StringVal("supersecret"),
									}),
								}),
								"credentials_helper": cty.MapVal(map[string]cty.Value{
									"credstore": cty.ObjectVal(map[string]cty.Value{
										"args": cty.ListVal([]cty.Value{cty.StringVal("--host=credstore.example.com")}),
									}),
								}),
								"provider_installation": cty.ListVal([]cty.Value{
									cty.ObjectVal(map[string]cty.Value{
										"dev_overrides": cty.MapVal(map[string]cty.Value{
											"hashicorp/null": cty.StringVal("/home/developer/tmp/terraform-null"),
										}),
										"filesystem_mirror": cty.ListVal([]cty.Value{
											cty.ObjectVal(map[string]cty.Value{
												"path": cty.StringVal("/usr/share/terraform/providers"),
												"include": cty.ListVal([]cty.Value{
													cty.StringVal("examplef.com/*/*"),
												}),
												"exclude": cty.NullVal(cty.List(cty.String)),
											}),
										}),
										"network_mirror": cty.ListVal([]cty.Value{
											cty.ObjectVal(map[string]cty.Value{
												"url": cty.StringVal("https://providers.example.com"),
												"include": cty.ListVal([]cty.Value{
													cty.StringVal("examplen.com/*/*"),
												}),
												"exclude": cty.NullVal(cty.List(cty.String)),
											}),
										}),
										"direct": cty.ListVal([]cty.Value{
											cty.ObjectVal(map[string]cty.Value{
												"include": cty.NullVal(cty.List(cty.String)),
												"exclude": cty.ListVal([]cty.Value{
													cty.StringVal("exampled.com/*/*"),
												}),
											}),
										}),
									}),
								}),
							}),
						},
						Steps: []*ScenarioStep{
							{
								Name: "backend",
								Module: &Module{
									Name:   "backend",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"driver": StepVariableVal(&StepVariable{Value: cty.StringVal("postgres")}),
									},
								},
							},
						},
					},
					{
						Name:         "default",
						TerraformCLI: DefaultTerraformCLI(),
						Steps: []*ScenarioStep{
							{
								Name: "backend",
								Module: &Module{
									Name:   "backend",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"driver": StepVariableVal(&StepVariable{Value: cty.StringVal("postgres")}),
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
