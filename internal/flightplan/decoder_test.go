package flightplan

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
)

func Test_Decoder_parseDir(t *testing.T) {
	t.Parallel()

	newDecoder := func(dir string) *Decoder {
		t.Helper()
		path, err := filepath.Abs(filepath.Join("./tests", dir))
		require.NoError(t, err)

		d, err := NewDecoder(
			WithDecoderBaseDir(path),
		)
		require.NoError(t, err)

		return d
	}

	t.Run("malformed enos.hcl", func(t *testing.T) {
		decoder := newDecoder("parse_dir_fail_malformed_config")
		diags := decoder.Parse()
		require.True(t, diags.HasErrors())
		require.Equal(t, hcl.DiagError, diags[0].Severity)
	})

	t.Run("no matching configuration files", func(t *testing.T) {
		decoder := newDecoder("parse_dir_pass_no_matching_names")
		diags := decoder.Parse()
		require.False(t, diags.HasErrors())
		require.Equal(t, 0, len(decoder.parser.Files()))
	})

	t.Run("two matching files", func(t *testing.T) {
		decoder := newDecoder("parse_dir_pass_two_matching_names")
		diags := decoder.Parse()
		require.False(t, diags.HasErrors())
		require.Equal(t, 2, len(decoder.parser.Files()))
	})
}

func Test_Decoder_Decode(t *testing.T) {
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
			desc: "simple scenario",
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "basic" {
  step "first" {
    module = module.backend
  }
}
`, modulePath),
			expected: &FlightPlan{
				Modules: []*Module{
					{
						Name:   "backend",
						Source: modulePath,
					},
				},
				Scenarios: []*Scenario{
					{
						Name: "basic",
						Steps: []*ScenarioStep{
							{
								Name: "first",
								Module: &ScenarioStepModule{
									Name:   "backend",
									Source: modulePath,
									Attrs:  map[string]cty.Value{},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "modules from the registry",
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
				Modules: []*Module{
					{
						Name:    "backend",
						Source:  "terraform-aws-modules/vpc/aws",
						Version: "3.11.0",
					},
				},
				Scenarios: []*Scenario{
					{
						Name: "basic",
						Steps: []*ScenarioStep{
							{
								Name: "first",
								Module: &ScenarioStepModule{
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
		{
			desc: "module references",
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
    module = module.frontend_blue
  }

  step "frontend_green" {
    module = module.frontend_green
  }

  step "frontend_red" {
    module = module.frontend_red
  }
}
`, modulePath),
			expected: &FlightPlan{
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
						Name: "basic",
						Steps: []*ScenarioStep{
							{
								Name: "backend",
								Module: &ScenarioStepModule{
									Name:   "backend",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"driver": cty.StringVal("postgres"),
									},
								},
							},
							{
								Name: "frontend_blue",
								Module: &ScenarioStepModule{
									Name:   "frontend_blue",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"app_version": cty.StringVal("1.0.0"),
									},
								},
							},
							{
								Name: "frontend_green",
								Module: &ScenarioStepModule{
									Name:   "frontend_green",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"app_version": cty.StringVal("1.1.0"),
									},
								},
							},
							{
								Name: "frontend_red",
								Module: &ScenarioStepModule{
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
			desc: "invalid enos identifier module block",
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
			desc: "invalid enos identifier scenario step block",
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
			desc: "invalid block in flight plan",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

notablock "something" {
  something = "else"
}

scenario "backend" {
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
			desc: "invalid block in scenario step",
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
			desc: "invalid attr in flightplan",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

notanattr = "foo"

scenario "backend" {
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
			desc: "invalid attr in scenario step",
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
			desc: "count meta-arg attr in module",
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
			desc: "for_each meta-arg attr in module",
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
			desc: "depends_on meta-arg attr in module",
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
		{
			desc: "count meta-arg attr in step variables",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    module = module.backend
    variables = {
      count = 1
    }
  }
}
`, modulePath),
		},
		{
			desc: "for_each meta-arg attr in step variables",
			fail: true,
			hcl: fmt.Sprintf(`
module "backend" {
  source = "%s"
}

scenario "backend" {
  step "first" {
    variables = {
      for_each = toset(["1", "2"])
    }
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "depends_on meta-arg attr in step variables",
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
    variables = {
      depends_on = module.backend
    }
    module = module.backend
  }
}
`, modulePath),
		},
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
			desc: "redeclared step in scenario",
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
	} {
		t.Run(test.desc, func(t *testing.T) {
			cwd, err := os.Getwd()
			require.NoError(t, err)
			decoder, err := NewDecoder(WithDecoderBaseDir(cwd))
			require.NoError(t, err)
			diags := decoder.parseHCL([]byte(test.hcl), "decoder-test.hcl")
			require.False(t, diags.HasErrors(), diags.Error())

			fp, moreDiags := decoder.Decode()
			if test.fail {
				require.True(t, moreDiags.HasErrors(), moreDiags.Error())
			} else {
				require.False(t, moreDiags.HasErrors(), moreDiags.Error())

				require.Len(t, fp.Modules, len(test.expected.Modules))
				require.Len(t, fp.Scenarios, len(test.expected.Scenarios))

				for i := range test.expected.Modules {
					require.EqualValues(t, test.expected.Modules[i].Name, fp.Modules[i].Name)
					require.EqualValues(t, test.expected.Modules[i].Source, fp.Modules[i].Source)
					require.EqualValues(t, test.expected.Modules[i].Version, fp.Modules[i].Version)
					if len(test.expected.Modules[i].Attrs) > 0 {
						assert.EqualValues(t, test.expected.Modules[i].Attrs, fp.Modules[i].Attrs)
					}
				}

				for i := range test.expected.Scenarios {
					require.EqualValues(t, test.expected.Scenarios[i].Name, fp.Scenarios[i].Name)
					require.EqualValues(t, test.expected.Scenarios[i].Steps, fp.Scenarios[i].Steps)
				}
			}
		})
	}
}
