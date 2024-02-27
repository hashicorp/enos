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

// Test_Decode_TerraformSettings.
func Test_Decode_TerraformSettings(t *testing.T) {
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
terraform ":hascolon" {
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
terraform "debug" {
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
terraform "debug" {
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
terraform "debug" {
  required_version = ">= 1.0.0"
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
  terraform = terraform.bad_ref

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "invalid more than one backend",
			fail: true,
			hcl: fmt.Sprintf(`
terraform "debug" {
  backend "remote" {
    workspaces {
      name = "enos"
    }
  }

  backend "consul" {
    address = "consul.example.com"
    scheme  = "https"
    path    = "full/path"
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
  terraform = terraform.bad_ref

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "invalid backend and cloud both defined",
			fail: true,
			hcl: fmt.Sprintf(`
terraform "debug" {
  backend "remote" {
    hostname = "remote.terraform.io"

    workspaces {
      name = "enos"
    }
  }

  cloud {
    organization = "qti"
    hostname = "cloud.terraform.io"
    token = "whyunouselogin"

    workspaces {
      name = "foo"
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
  terraform = terraform.bad_ref

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "maximal with backend configuration",
			hcl: fmt.Sprintf(`
terraform "default" {
  required_version = ">= 1.1.0"
  experiments      = ["something"]

  required_providers {
    aws = {
      version = ">= 2.7.0"
      source = "hashicorp/aws"
    }
  }

  provider_meta "enos" {
    hello = "world"
  }

  backend "remote" {
    // A whole bunch of attrs depending on which backend we're using
    organization = "mightberequired"

    workspaces {
      name = "enos"
      // OR
      prefix = "enos-"
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
  terraform = terraform.default

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				TerraformSettings: []*TerraformSetting{
					{
						Name:            "default",
						RequiredVersion: cty.StringVal(">= 1.1.0"),
						Experiments:     cty.ListVal([]cty.Value{cty.StringVal("something")}),
						RequiredProviders: map[string]cty.Value{
							"aws": cty.ObjectVal(map[string]cty.Value{
								"version": cty.StringVal(">= 2.7.0"),
								"source":  cty.StringVal("hashicorp/aws"),
							}),
						},
						ProviderMetas: map[string]map[string]cty.Value{
							"enos": {
								"hello": cty.StringVal("world"),
							},
						},
						Backend: &TerraformSettingBackend{
							Name: "remote",
							Attrs: map[string]cty.Value{
								"organization": cty.StringVal("mightberequired"),
							},
							Workspaces: cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
								"name":   cty.StringVal("enos"),
								"prefix": cty.StringVal("enos-"),
							})}),
						},
						Cloud: cty.ObjectVal(map[string]cty.Value{
							"cloud": cty.ListValEmpty(cty.Object(map[string]cty.Type{
								"hostname":     cty.String,
								"organization": cty.String,
								"token":        cty.String,
								"workspaces": cty.List(cty.Object(map[string]cty.Type{
									"name": cty.String,
									"tags": cty.List(cty.String),
								})),
							})),
						}),
					},
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
				ScenarioBlocks: DecodedScenarioBlocks{
					{
						Name: "debug",
						Scenarios: []*Scenario{
							{
								Name:         "debug",
								TerraformCLI: DefaultTerraformCLI(),
								TerraformSetting: &TerraformSetting{
									Name:            "default",
									RequiredVersion: cty.StringVal(">= 1.1.0"),
									Experiments:     cty.ListVal([]cty.Value{cty.StringVal("something")}),
									RequiredProviders: map[string]cty.Value{
										"aws": cty.ObjectVal(map[string]cty.Value{
											"version": cty.StringVal(">= 2.7.0"),
											"source":  cty.StringVal("hashicorp/aws"),
										}),
									},
									ProviderMetas: map[string]map[string]cty.Value{
										"enos": {
											"hello": cty.StringVal("world"),
										},
									},
									Backend: &TerraformSettingBackend{
										Name: "remote",
										Attrs: map[string]cty.Value{
											"organization": cty.StringVal("mightberequired"),
										},
										Workspaces: cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
											"name":   cty.StringVal("enos"),
											"prefix": cty.StringVal("enos-"),
										})}),
									},
									Cloud: cty.ObjectVal(map[string]cty.Value{
										"cloud": cty.ListValEmpty(cty.Object(map[string]cty.Type{
											"hostname":     cty.String,
											"organization": cty.String,
											"token":        cty.String,
											"workspaces": cty.List(cty.Object(map[string]cty.Type{
												"name": cty.String,
												"tags": cty.List(cty.String),
											})),
										})),
									}),
								},
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
								},
							},
						},
					},
					{
						Name: "default",
						Scenarios: []*Scenario{
							{
								Name:         "default",
								TerraformCLI: DefaultTerraformCLI(),
								TerraformSetting: &TerraformSetting{
									Name:            "default",
									RequiredVersion: cty.StringVal(">= 1.1.0"),
									Experiments:     cty.ListVal([]cty.Value{cty.StringVal("something")}),
									RequiredProviders: map[string]cty.Value{
										"aws": cty.ObjectVal(map[string]cty.Value{
											"version": cty.StringVal(">= 2.7.0"),
											"source":  cty.StringVal("hashicorp/aws"),
										}),
									},
									ProviderMetas: map[string]map[string]cty.Value{
										"enos": {
											"hello": cty.StringVal("world"),
										},
									},
									Backend: &TerraformSettingBackend{
										Name: "remote",
										Attrs: map[string]cty.Value{
											"organization": cty.StringVal("mightberequired"),
										},
										Workspaces: cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
											"name":   cty.StringVal("enos"),
											"prefix": cty.StringVal("enos-"),
										})}),
									},
									Cloud: cty.ObjectVal(map[string]cty.Value{
										"cloud": cty.ListValEmpty(cty.Object(map[string]cty.Type{
											"hostname":     cty.String,
											"organization": cty.String,
											"token":        cty.String,
											"workspaces": cty.List(cty.Object(map[string]cty.Type{
												"name": cty.String,
												"tags": cty.List(cty.String),
											})),
										})),
									}),
								},
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
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "maximal with cloud configuration",
			hcl: fmt.Sprintf(`
terraform "default" {
  required_version = ">= 1.1.0"
  experiments      = ["something"]

  required_providers {
    aws = {
      version = ">= 2.7.0"
      source = "hashicorp/aws"
    }
  }

  provider_meta "enos" {
    hello = "world"
  }

  cloud {
    organization = "qti"
    hostname = "cloud.terraform.io"
    token = "yunouselogin"

    workspaces {
      tags = ["something", "another"]
      name = "foo"
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
  terraform = terraform.default

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				TerraformSettings: []*TerraformSetting{
					{
						Name:            "default",
						RequiredVersion: cty.StringVal(">= 1.1.0"),
						Experiments:     cty.ListVal([]cty.Value{cty.StringVal("something")}),
						RequiredProviders: map[string]cty.Value{
							"aws": cty.ObjectVal(map[string]cty.Value{
								"version": cty.StringVal(">= 2.7.0"),
								"source":  cty.StringVal("hashicorp/aws"),
							}),
						},
						ProviderMetas: map[string]map[string]cty.Value{
							"enos": {
								"hello": cty.StringVal("world"),
							},
						},
						Backend: nil,
						Cloud: cty.ObjectVal(map[string]cty.Value{
							"cloud": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
								"hostname":     cty.StringVal("cloud.terraform.io"),
								"organization": cty.StringVal("qti"),
								"token":        cty.StringVal("yunouselogin"),
								"workspaces": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
									"tags": cty.ListVal([]cty.Value{
										cty.StringVal("something"),
										cty.StringVal("another"),
									}),
									"name": cty.StringVal("foo"),
								})}),
							})}),
						}),
					},
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
				ScenarioBlocks: DecodedScenarioBlocks{
					{
						Name: "debug",
						Scenarios: []*Scenario{
							{
								Name:         "debug",
								TerraformCLI: DefaultTerraformCLI(),
								TerraformSetting: &TerraformSetting{
									Name:            "default",
									RequiredVersion: cty.StringVal(">= 1.1.0"),
									Experiments:     cty.ListVal([]cty.Value{cty.StringVal("something")}),
									RequiredProviders: map[string]cty.Value{
										"aws": cty.ObjectVal(map[string]cty.Value{
											"version": cty.StringVal(">= 2.7.0"),
											"source":  cty.StringVal("hashicorp/aws"),
										}),
									},
									ProviderMetas: map[string]map[string]cty.Value{
										"enos": {
											"hello": cty.StringVal("world"),
										},
									},
									Backend: NewTerraformSettingBackend(),
									Cloud: cty.ObjectVal(map[string]cty.Value{
										"cloud": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
											"hostname":     cty.StringVal("cloud.terraform.io"),
											"organization": cty.StringVal("qti"),
											"token":        cty.StringVal("yunouselogin"),
											"workspaces": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
												"tags": cty.ListVal([]cty.Value{
													cty.StringVal("something"),
													cty.StringVal("another"),
												}),
												"name": cty.StringVal("foo"),
											})}),
										})}),
									}),
								},
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
								},
							},
						},
					},
					{
						Name: "default",
						Scenarios: []*Scenario{
							{
								Name:         "default",
								TerraformCLI: DefaultTerraformCLI(),
								TerraformSetting: &TerraformSetting{
									Name:            "default",
									RequiredVersion: cty.StringVal(">= 1.1.0"),
									Experiments:     cty.ListVal([]cty.Value{cty.StringVal("something")}),
									RequiredProviders: map[string]cty.Value{
										"aws": cty.ObjectVal(map[string]cty.Value{
											"version": cty.StringVal(">= 2.7.0"),
											"source":  cty.StringVal("hashicorp/aws"),
										}),
									},
									ProviderMetas: map[string]map[string]cty.Value{
										"enos": {
											"hello": cty.StringVal("world"),
										},
									},
									Backend: NewTerraformSettingBackend(),
									Cloud: cty.ObjectVal(map[string]cty.Value{
										"cloud": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
											"hostname":     cty.StringVal("cloud.terraform.io"),
											"organization": cty.StringVal("qti"),
											"token":        cty.StringVal("yunouselogin"),
											"workspaces": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
												"tags": cty.ListVal([]cty.Value{
													cty.StringVal("something"),
													cty.StringVal("another"),
												}),
												"name": cty.StringVal("foo"),
											})}),
										})}),
									}),
								},
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
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "inherits default terraform setting",
			hcl: fmt.Sprintf(`
terraform "default" {
  cloud {
    organization = "qti"
    hostname = "cloud.terraform.io"
    token = "yunouselogin"

    workspaces {
      tags = ["something", "another"]
      name = "foo"
    }
  }
}

terraform "experiments" {
  experiments = ["something"]

  cloud {
    organization = "qti"
    hostname = "cloud.terraform.io"
    token = "yunouselogin"

    workspaces {
      tags = ["something", "another"]
      name = "foo"
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

scenario "experiments" {
  terraform = terraform.experiments

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				TerraformSettings: []*TerraformSetting{
					{
						Name:              "default",
						RequiredVersion:   cty.NullVal(cty.String),
						Experiments:       cty.NullVal(cty.List(cty.String)),
						RequiredProviders: map[string]cty.Value{},
						ProviderMetas:     map[string]map[string]cty.Value{},
						Cloud: cty.ObjectVal(map[string]cty.Value{
							"cloud": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
								"hostname":     cty.StringVal("cloud.terraform.io"),
								"organization": cty.StringVal("qti"),
								"token":        cty.StringVal("yunouselogin"),
								"workspaces": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
									"tags": cty.ListVal([]cty.Value{
										cty.StringVal("something"),
										cty.StringVal("another"),
									}),
									"name": cty.StringVal("foo"),
								})}),
							})}),
						}),
					},
					{
						Name:              "experiments",
						RequiredVersion:   cty.NullVal(cty.String),
						Experiments:       cty.ListVal([]cty.Value{cty.StringVal("something")}),
						RequiredProviders: map[string]cty.Value{},
						ProviderMetas:     map[string]map[string]cty.Value{},
						Cloud: cty.ObjectVal(map[string]cty.Value{
							"cloud": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
								"hostname":     cty.StringVal("cloud.terraform.io"),
								"organization": cty.StringVal("qti"),
								"token":        cty.StringVal("yunouselogin"),
								"workspaces": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
									"tags": cty.ListVal([]cty.Value{
										cty.StringVal("something"),
										cty.StringVal("another"),
									}),
									"name": cty.StringVal("foo"),
								})}),
							})}),
						}),
					},
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
				ScenarioBlocks: DecodedScenarioBlocks{
					{
						Name: "default",
						Scenarios: []*Scenario{
							{
								Name:         "default",
								TerraformCLI: DefaultTerraformCLI(),
								TerraformSetting: &TerraformSetting{
									Name:              "default",
									RequiredVersion:   cty.NullVal(cty.String),
									Experiments:       cty.NullVal(cty.List(cty.String)),
									RequiredProviders: map[string]cty.Value{},
									ProviderMetas:     map[string]map[string]cty.Value{},
									Backend:           NewTerraformSettingBackend(),
									Cloud: cty.ObjectVal(map[string]cty.Value{
										"cloud": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
											"hostname":     cty.StringVal("cloud.terraform.io"),
											"organization": cty.StringVal("qti"),
											"token":        cty.StringVal("yunouselogin"),
											"workspaces": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
												"tags": cty.ListVal([]cty.Value{
													cty.StringVal("something"),
													cty.StringVal("another"),
												}),
												"name": cty.StringVal("foo"),
											})}),
										})}),
									}),
								},
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
								},
							},
						},
					},
					{
						Name: "experiments",
						Scenarios: []*Scenario{
							{
								Name:         "experiments",
								TerraformCLI: DefaultTerraformCLI(),
								TerraformSetting: &TerraformSetting{
									Name:              "experiments",
									RequiredVersion:   cty.NullVal(cty.String),
									Experiments:       cty.ListVal([]cty.Value{cty.StringVal("something")}),
									RequiredProviders: map[string]cty.Value{},
									ProviderMetas:     map[string]map[string]cty.Value{},
									Backend:           NewTerraformSettingBackend(),
									Cloud: cty.ObjectVal(map[string]cty.Value{
										"cloud": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
											"hostname":     cty.StringVal("cloud.terraform.io"),
											"organization": cty.StringVal("qti"),
											"token":        cty.StringVal("yunouselogin"),
											"workspaces": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
												"tags": cty.ListVal([]cty.Value{
													cty.StringVal("something"),
													cty.StringVal("another"),
												}),
												"name": cty.StringVal("foo"),
											})}),
										})}),
									}),
								},
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
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "allows string based address references",
			hcl: fmt.Sprintf(`
terraform "experiments" {
  experiments = ["something"]

  cloud {
    organization = "qti"
    hostname = "cloud.terraform.io"
    token = "yunouselogin"

    workspaces {
      tags = ["something", "another"]
      name = "foo"
    }
  }
}

module "backend" {
  source = "%s"

  driver = "postgres"
}

scenario "default" {
  terraform = "experiments"
  step "backend" {
    module = module.backend
  }
}
`, modulePath),
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				TerraformSettings: []*TerraformSetting{
					{
						Name:              "experiments",
						RequiredVersion:   cty.NullVal(cty.String),
						Experiments:       cty.ListVal([]cty.Value{cty.StringVal("something")}),
						RequiredProviders: map[string]cty.Value{},
						ProviderMetas:     map[string]map[string]cty.Value{},
						Cloud: cty.ObjectVal(map[string]cty.Value{
							"cloud": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
								"hostname":     cty.StringVal("cloud.terraform.io"),
								"organization": cty.StringVal("qti"),
								"token":        cty.StringVal("yunouselogin"),
								"workspaces": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
									"tags": cty.ListVal([]cty.Value{
										cty.StringVal("something"),
										cty.StringVal("another"),
									}),
									"name": cty.StringVal("foo"),
								})}),
							})}),
						}),
					},
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
				ScenarioBlocks: DecodedScenarioBlocks{
					{
						Name: "default",
						Scenarios: []*Scenario{
							{
								Name:         "default",
								TerraformCLI: DefaultTerraformCLI(),
								TerraformSetting: &TerraformSetting{
									Name:              "experiments",
									RequiredVersion:   cty.NullVal(cty.String),
									Experiments:       cty.ListVal([]cty.Value{cty.StringVal("something")}),
									RequiredProviders: map[string]cty.Value{},
									ProviderMetas:     map[string]map[string]cty.Value{},
									Backend:           NewTerraformSettingBackend(),
									Cloud: cty.ObjectVal(map[string]cty.Value{
										"cloud": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
											"hostname":     cty.StringVal("cloud.terraform.io"),
											"organization": cty.StringVal("qti"),
											"token":        cty.StringVal("yunouselogin"),
											"workspaces": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
												"tags": cty.ListVal([]cty.Value{
													cty.StringVal("something"),
													cty.StringVal("another"),
												}),
												"name": cty.StringVal("foo"),
											})}),
										})}),
									}),
								},
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

func Test_TerraformSettings_Cty_RoundTrip(t *testing.T) {
	t.Parallel()

	setting := &TerraformSetting{
		Name:            "default",
		RequiredVersion: cty.StringVal(">= 1.1.0"),
		Experiments:     cty.ListVal([]cty.Value{cty.StringVal("something")}),
		RequiredProviders: map[string]cty.Value{
			"aws": cty.ObjectVal(map[string]cty.Value{
				"version": cty.StringVal(">= 2.7.0"),
				"source":  cty.StringVal("hashicorp/aws"),
			}),
		},
		ProviderMetas: map[string]map[string]cty.Value{
			"enos": {
				"hello": cty.StringVal("world"),
			},
		},
		Backend: &TerraformSettingBackend{
			Name: "remote",
			Attrs: map[string]cty.Value{
				"organization": cty.StringVal("mightberequired"),
			},
			Workspaces: cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"name":   cty.StringVal("enos"),
				"prefix": cty.StringVal("enos-"),
			})}),
		},
		Cloud: cty.ObjectVal(map[string]cty.Value{
			"cloud": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.StringVal("cloud.terraform.io"),
				"organization": cty.StringVal("qti"),
				"token":        cty.StringVal("yunouselogin"),
				"workspaces": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"tags": cty.ListVal([]cty.Value{
						cty.StringVal("something"),
						cty.StringVal("another"),
					}),
					"name": cty.StringVal("foo"),
				})}),
			})}),
		}),
	}

	clone := NewTerraformSetting()
	require.NoError(t, clone.FromCtyValue(setting.ToCtyValue()))
	require.EqualValues(t, setting, clone)
}
