package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test_Decode_Provider tests transport.
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
						Config: &SchemalessBlock{
							Type:   "provider",
							Labels: []string{"enos", "ubuntu"},
							Attrs: map[string]cty.Value{
								"transport": cty.ObjectVal(map[string]cty.Value{
									"ssh": cty.ObjectVal(map[string]cty.Value{
										"user":        cty.StringVal("ubuntu"),
										"private_key": cty.StringVal("supersecret"),
									}),
								}),
							},
							Children: []*SchemalessBlock{},
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
				ScenarioBlocks: DecodedScenarioBlocks{
					{
						Name: "test",
						Scenarios: []*Scenario{
							{
								Name:         "test",
								TerraformCLI: DefaultTerraformCLI(),
								Providers: []*Provider{
									{
										Type:  "enos",
										Alias: "ubuntu",
										Config: &SchemalessBlock{
											Type:   "provider",
											Labels: []string{"enos", "ubuntu"},
											Attrs: map[string]cty.Value{
												"transport": cty.ObjectVal(map[string]cty.Value{
													"ssh": cty.ObjectVal(map[string]cty.Value{
														"user":        cty.StringVal("ubuntu"),
														"private_key": cty.StringVal("supersecret"),
													}),
												}),
											},
											Children: []*SchemalessBlock{},
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
												Config: &SchemalessBlock{
													Type:   "provider",
													Labels: []string{"enos", "ubuntu"},
													Attrs: map[string]cty.Value{
														"transport": cty.ObjectVal(map[string]cty.Value{
															"ssh": cty.ObjectVal(map[string]cty.Value{
																"user":        cty.StringVal("ubuntu"),
																"private_key": cty.StringVal("supersecret"),
															}),
														}),
													},
													Children: []*SchemalessBlock{},
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

provider "kubernetes" "default" {
  host = "eks.host.com"
  cluster_ca_certificate = "base64cert"
  exec {
    api_version = "client.authentication.k8s.io/v1alpha1"
    args        = ["eks", "get-token", "--cluster-name", "my-cluster"]
    command     = "aws"

    not_a_real_block_but_testing_nested_things {
      nested_attr = "value"
    }
  }
}

module "copy" {
  source = "%s"

  driver = "s3"
}

module "k8s_deploy" {
  source = "%[1]s"
  driver = "k8s"
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

scenario "k8s" {
  providers = [
    provider.kubernetes.default,
  ]

  step "deploy" {
    module = module.k8s_deploy
  }
}`, modulePath),
			expected: &FlightPlan{
				Providers: []*Provider{
					{
						Type:  "aws",
						Alias: "east",
						Config: &SchemalessBlock{
							Type:   "provider",
							Labels: []string{"aws", "east"},
							Attrs: map[string]cty.Value{
								"region": cty.StringVal("us-east-1"),
							},
							Children: []*SchemalessBlock{},
						},
					},
					{
						Type:  "aws",
						Alias: "west",
						Config: &SchemalessBlock{
							Type:   "provider",
							Labels: []string{"aws", "west"},
							Attrs: map[string]cty.Value{
								"region": cty.StringVal("us-west-1"),
							},
							Children: []*SchemalessBlock{},
						},
					},
					{
						Type:  "aws",
						Alias: "eu",
						Config: &SchemalessBlock{
							Type:   "provider",
							Labels: []string{"aws", "eu"},
							Attrs: map[string]cty.Value{
								"region": cty.StringVal("eu-west-1"),
							},
							Children: []*SchemalessBlock{},
						},
					},
					{
						Type:  "kubernetes",
						Alias: "default",
						Config: &SchemalessBlock{
							Type:   "provider",
							Labels: []string{"kubernetes", "default"},
							Attrs: map[string]cty.Value{
								"host":                   cty.StringVal("eks.host.com"),
								"cluster_ca_certificate": cty.StringVal("base64cert"),
							},
							Children: []*SchemalessBlock{
								{
									Type:   "exec",
									Labels: []string{},
									Attrs: map[string]cty.Value{
										"api_version": cty.StringVal("client.authentication.k8s.io/v1alpha1"),
										"args": cty.ListVal([]cty.Value{
											cty.StringVal("eks"),
											cty.StringVal("get-token"),
											cty.StringVal("--cluster-name"),
											cty.StringVal("my-cluster"),
										}),
										"command": cty.StringVal("aws"),
									},
									Children: []*SchemalessBlock{
										{
											Type:   "not_a_real_block_but_testing_nested_things",
											Labels: []string{"with", "labels"},
											Attrs: map[string]cty.Value{
												"nested_attr": cty.StringVal("value"),
											},
										},
									},
								},
							},
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
					{
						Name:   "k8s_deploy",
						Source: modulePath,
						Attrs: map[string]cty.Value{
							"driver": cty.StringVal("k8s"),
						},
					},
				},
				ScenarioBlocks: DecodedScenarioBlocks{
					{
						Name: "copy_to_east",
						Scenarios: []*Scenario{
							{
								Name:         "copy_to_east",
								TerraformCLI: DefaultTerraformCLI(),
								Providers: []*Provider{
									{
										Type:  "aws",
										Alias: "west",
										Config: &SchemalessBlock{
											Type:   "provider",
											Labels: []string{"aws", "west"},
											Attrs: map[string]cty.Value{
												"region": cty.StringVal("us-west-1"),
											},
											Children: []*SchemalessBlock{},
										},
									},
									{
										Type:  "aws",
										Alias: "east",
										Config: &SchemalessBlock{
											Type:   "provider",
											Labels: []string{"aws", "east"},
											Attrs: map[string]cty.Value{
												"region": cty.StringVal("us-east-1"),
											},
											Children: []*SchemalessBlock{},
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
												Config: &SchemalessBlock{
													Type:   "provider",
													Labels: []string{"aws", "west"},
													Attrs: map[string]cty.Value{
														"region": cty.StringVal("us-west-1"),
													},
													Children: []*SchemalessBlock{},
												},
											},
											"dst": {
												Type:  "aws",
												Alias: "east",
												Config: &SchemalessBlock{
													Type:   "provider",
													Labels: []string{"aws", "east"},
													Attrs: map[string]cty.Value{
														"region": cty.StringVal("us-east-1"),
													},
													Children: []*SchemalessBlock{},
												},
											},
										},
									},
								},
							},
						},
					},
					{
						Name: "copy_to_eu",
						Scenarios: []*Scenario{
							{
								Name:         "copy_to_eu",
								TerraformCLI: DefaultTerraformCLI(),
								Providers: []*Provider{
									{
										Type:  "aws",
										Alias: "east",
										Config: &SchemalessBlock{
											Type:   "provider",
											Labels: []string{"aws", "east"},
											Attrs: map[string]cty.Value{
												"region": cty.StringVal("us-east-1"),
											},
											Children: []*SchemalessBlock{},
										},
									},
									{
										Type:  "aws",
										Alias: "eu",
										Config: &SchemalessBlock{
											Type:   "provider",
											Labels: []string{"aws", "eu"},
											Attrs: map[string]cty.Value{
												"region": cty.StringVal("eu-west-1"),
											},
											Children: []*SchemalessBlock{},
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
												Config: &SchemalessBlock{
													Type:   "provider",
													Labels: []string{"aws", "east"},
													Attrs: map[string]cty.Value{
														"region": cty.StringVal("us-east-1"),
													},
													Children: []*SchemalessBlock{},
												},
											},
											"dst": {
												Type:  "aws",
												Alias: "eu",
												Config: &SchemalessBlock{
													Type:   "provider",
													Labels: []string{"aws", "eu"},
													Attrs: map[string]cty.Value{
														"region": cty.StringVal("eu-west-1"),
													},
													Children: []*SchemalessBlock{},
												},
											},
										},
									},
								},
							},
						},
					},
					{
						Name: "k8s",
						Scenarios: []*Scenario{
							{
								Name:         "k8s",
								TerraformCLI: DefaultTerraformCLI(),
								Providers: []*Provider{
									{
										Type:  "kubernetes",
										Alias: "default",
										Config: &SchemalessBlock{
											Type:   "provider",
											Labels: []string{"kubernetes", "default"},
											Attrs: map[string]cty.Value{
												"host":                   cty.StringVal("eks.host.com"),
												"cluster_ca_certificate": cty.StringVal("base64cert"),
											},
											Children: []*SchemalessBlock{
												{
													Type:   "exec",
													Labels: []string{},
													Attrs: map[string]cty.Value{
														"api_version": cty.StringVal("client.authentication.k8s.io/v1alpha1"),
														"args": cty.ListVal([]cty.Value{
															cty.StringVal("eks"),
															cty.StringVal("get-token"),
															cty.StringVal("--cluster-name"),
															cty.StringVal("my-cluster"),
														}),
														"command": cty.StringVal("aws"),
													},
													Children: []*SchemalessBlock{
														{
															Type:   "not_a_real_block_but_testing_nested_things",
															Labels: []string{"with", "labels"},
															Attrs: map[string]cty.Value{
																"nested_attr": cty.StringVal("value"),
															},
															Children: []*SchemalessBlock{},
														},
													},
												},
											},
										},
									},
								},
								Steps: []*ScenarioStep{
									{
										Name: "deploy",
										Module: &Module{
											Name:   "k8s_deploy",
											Source: modulePath,
											Attrs: map[string]cty.Value{
												"driver": testMakeStepVarValue(cty.StringVal("k8s")),
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
		test := test
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

func Test_Provider_Cty_RoundTrip(t *testing.T) {
	t.Parallel()

	for _, p := range []struct {
		testName string
		provider *Provider
	}{
		{
			"aws",
			&Provider{
				Type:  "aws",
				Alias: "west",
				Config: &SchemalessBlock{
					Labels: []string{},
					Attrs: map[string]cty.Value{
						"region":     cty.StringVal("us-west-1"),
						"access_key": cty.StringVal("key"),
						"secret_key": cty.StringVal("secret"),
					},
					Children: []*SchemalessBlock{},
				},
			},
		},
		{
			"k8s",
			&Provider{
				Type:  "kubernetes",
				Alias: "default",
				Config: &SchemalessBlock{
					Type:   "provider",
					Labels: []string{"kubernetes", "default"},
					Attrs: map[string]cty.Value{
						"host":                   cty.StringVal("eks.host.com"),
						"cluster_ca_certificate": cty.StringVal("base64cert"),
					},
					Children: []*SchemalessBlock{
						{
							Type:   "exec",
							Labels: []string{},
							Attrs: map[string]cty.Value{
								"api_version": cty.StringVal("client.authentication.k8s.io/v1alpha1"),
								"args": cty.ListVal([]cty.Value{
									cty.StringVal("eks"),
									cty.StringVal("get-token"),
									cty.StringVal("--cluster-name"),
									cty.StringVal("my-cluster"),
								}),
								"command": cty.StringVal("aws"),
							},
							Children: []*SchemalessBlock{
								{
									Type:   "not_a_real_block_but_testing_nested_things",
									Labels: []string{"with", "labels"},
									Attrs: map[string]cty.Value{
										"nested_attr": cty.StringVal("value"),
									},
									Children: []*SchemalessBlock{},
								},
							},
						},
					},
				},
			},
		},
	} {
		p := p
		t.Run(p.testName, func(t *testing.T) {
			t.Parallel()

			clone := NewProvider()
			require.NoError(t, clone.FromCtyValue(p.provider.ToCtyValue()))
			require.EqualValues(t, p.provider, clone)
		})
	}
}
