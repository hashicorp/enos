package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test_Decode_Provider tests transport
func Test_Decode_Provider(t *testing.T) {
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
provider ":hascolon" "something" {
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
provider "foo" "bar" {
  invalid_block "foo" {
    foo = "bar"
  }
}

module "backend" {
  source = "%s"
}

scenario "backend" {
  providers = [provider.foo.bar]

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
provider "aws" "east" {
  region = "us-east-1"
}

module "backend" {
  source = "%s"

  driver = "postgres"
}

scenario "test" {
  providers = [provider.aws.west]

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "invalid step reference",
			fail: true,
			hcl: fmt.Sprintf(`
provider "aws" "east" {
  region = "us-east-1"
}

module "backend" {
  source = "%s"

  driver = "postgres"
}

scenario "test" {
  providers = [provider.aws.west]

  step "backend" {
    module    = module.backend
    providers = {
      aws = provider.aws.east
    }
  }
}
`, modulePath),
		},
		{
			desc: "enos transport",
			hcl: fmt.Sprintf(`
provider "enos" "ubuntu" {
  transport = {
    ssh = {
      user        = "ubuntu"
      private_key = "supersecret"
    }
  }
}

module "test" {
  source = "%s"
  driver = "s3"
}

scenario "test" {
  providers = [provider.enos.ubuntu]

  step "test" {
    module    = module.test
    providers = {
      enos = provider.enos.ubuntu
    }
  }
}
`, modulePath),
			expected: &FlightPlan{
				Providers: []*Provider{
					{
						Type:  "enos",
						Alias: "ubuntu",
						Attrs: map[string]cty.Value{
							"transport": cty.ObjectVal(map[string]cty.Value{
								"ssh": cty.ObjectVal(map[string]cty.Value{
									"user":        cty.StringVal("ubuntu"),
									"private_key": cty.StringVal("supersecret"),
								}),
							}),
						},
					},
				},
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				Modules: []*Module{
					{
						Name:   "test",
						Source: modulePath,
						Attrs: map[string]cty.Value{
							"driver": cty.StringVal("s3"),
						},
					},
				},
				Scenarios: []*Scenario{
					{
						Name:         "test",
						TerraformCLI: DefaultTerraformCLI(),
						Providers: []*Provider{
							{
								Type:  "enos",
								Alias: "ubuntu",
								Attrs: map[string]cty.Value{
									"transport": cty.ObjectVal(map[string]cty.Value{
										"ssh": cty.ObjectVal(map[string]cty.Value{
											"user":        cty.StringVal("ubuntu"),
											"private_key": cty.StringVal("supersecret"),
										}),
									}),
								},
							},
						},
						Steps: []*ScenarioStep{
							{
								Name: "test",
								Module: &Module{
									Name:   "test",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"driver": testMakeStepVarValue(cty.StringVal("s3")),
									},
								},
								Providers: map[string]*Provider{
									"enos": {
										Type:  "enos",
										Alias: "ubuntu",
										Attrs: map[string]cty.Value{
											"transport": cty.ObjectVal(map[string]cty.Value{
												"ssh": cty.ObjectVal(map[string]cty.Value{
													"user":        cty.StringVal("ubuntu"),
													"private_key": cty.StringVal("supersecret"),
												}),
											}),
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
			desc: "maximal configuration",
			hcl: fmt.Sprintf(`
provider "aws" "east" {
  region = "us-east-1"
}

provider "aws" "west" {
  region = "us-west-1"
}

provider "aws" "eu" {
  region = "eu-west-1"
}

module "copy" {
  source = "%s"

  driver = "s3"
}

scenario "copy_to_east" {
  providers = [
    provider.aws.west,
	"aws.east",
  ]

  step "copy" {
    module    = module.copy
    providers = {
      src = "aws.west"
      dst = provider.aws.east
    }
  }
}

scenario "copy_to_eu" {
  providers = [
    provider.aws.east,
    provider.aws.eu
  ]

  step "copy" {
    module    = module.copy
    providers = {
      src = provider.aws.east
      dst = provider.aws.eu
    }
  }
}
`, modulePath),
			expected: &FlightPlan{
				Providers: []*Provider{
					{
						Type: "aws",

						Alias: "east",
						Attrs: map[string]cty.Value{
							"region": cty.StringVal("us-east-1"),
						},
					},
					{
						Type:  "aws",
						Alias: "west",
						Attrs: map[string]cty.Value{
							"region": cty.StringVal("us-west-1"),
						},
					},
					{
						Type:  "aws",
						Alias: "eu",
						Attrs: map[string]cty.Value{
							"region": cty.StringVal("eu-west-1"),
						},
					},
				},
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				Modules: []*Module{
					{
						Name:   "copy",
						Source: modulePath,
						Attrs: map[string]cty.Value{
							"driver": cty.StringVal("s3"),
						},
					},
				},
				Scenarios: []*Scenario{
					{
						Name:         "copy_to_east",
						TerraformCLI: DefaultTerraformCLI(),
						Providers: []*Provider{
							{
								Type:  "aws",
								Alias: "east",
								Attrs: map[string]cty.Value{
									"region": cty.StringVal("us-east-1"),
								},
							},
							{
								Type:  "aws",
								Alias: "west",
								Attrs: map[string]cty.Value{
									"region": cty.StringVal("us-west-1"),
								},
							},
						},
						Steps: []*ScenarioStep{
							{
								Name: "copy",
								Module: &Module{
									Name:   "copy",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"driver": testMakeStepVarValue(cty.StringVal("s3")),
									},
								},
								Providers: map[string]*Provider{
									"src": {
										Type:  "aws",
										Alias: "west",
										Attrs: map[string]cty.Value{
											"region": cty.StringVal("us-west-1"),
										},
									},

									"dst": {
										Type:  "aws",
										Alias: "east",
										Attrs: map[string]cty.Value{
											"region": cty.StringVal("us-east-1"),
										},
									},
								},
							},
						},
					},
					{
						Name:         "copy_to_eu",
						TerraformCLI: DefaultTerraformCLI(),
						Providers: []*Provider{
							{
								Type:  "aws",
								Alias: "east",
								Attrs: map[string]cty.Value{
									"region": cty.StringVal("us-east-1"),
								},
							},
							{
								Type:  "aws",
								Alias: "eu",
								Attrs: map[string]cty.Value{
									"region": cty.StringVal("eu-west-1"),
								},
							},
						},
						Steps: []*ScenarioStep{
							{
								Name: "copy",
								Module: &Module{
									Name:   "copy",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"driver": testMakeStepVarValue(cty.StringVal("s3")),
									},
								},
								Providers: map[string]*Provider{
									"src": {
										Type:  "aws",
										Alias: "east",
										Attrs: map[string]cty.Value{
											"region": cty.StringVal("us-east-1"),
										},
									},
									"dst": {
										Type:  "aws",
										Alias: "eu",
										Attrs: map[string]cty.Value{
											"region": cty.StringVal("eu-west-1"),
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

func Test_Provider_Cty_RoundTrip(t *testing.T) {
	provider := &Provider{
		Type:  "aws",
		Alias: "west",
		Attrs: map[string]cty.Value{
			"region":     cty.StringVal("us-west-1"),
			"access_key": cty.StringVal("key"),
			"secret_key": cty.StringVal("secret"),
		},
	}

	clone := NewProvider()
	require.NoError(t, clone.FromCtyValue(provider.ToCtyValue()))
	require.EqualValues(t, provider, clone)
}
