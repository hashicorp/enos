package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test_Decode_Transport tests transport
func Test_Decode_Transport(t *testing.T) {
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
transport ":hascolon" {
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
transport "debug" {
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
enos_provider "debug" {
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
			desc: "invalid ssh attr",
			fail: true,
			hcl: fmt.Sprintf(`
enos_provider "debug" {
  ssh = {
    not_an_attr = "something"
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
			desc: "invalid reference",
			fail: true,
			hcl: fmt.Sprintf(`
transport "ubuntu" {
  ssh = {
    user     = "ubuntu"
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

scenario "ubuntu" {
  transport = transport.bad_ref

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
		},
		{
			desc: "maximal configuration",
			hcl: fmt.Sprintf(`
transport "ubuntu" {
  ssh = {
    user             = "ubuntu"
	host             = "192.168.0.1"
	private_key      = "...rsa private key..."
	private_key_path = "/tmp/private.key"
	passphrase        = "super secret passphrase"
    passphrase_path   = "/tmp/passphase"
  }
}

transport "default" {
  ssh = {
    user             = "ec2-user"
	host             = "192.168.0.2"
	private_key      = "...rsa private ec2 key..."
	private_key_path = "/tmp/ec2-private.key"
	passphrase        = "ec2 super secret passphrase"
    passphrase_path   = "/tmp/ec2-passphase"
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

scenario "ubuntu" {
  transport = transport.ubuntu

  step "backend" {
    module = module.backend
  }
}
`, modulePath),
			expected: &FlightPlan{
				Transports: []*Transport{
					{
						Name: "ubuntu",
						SSH: &TransportSSH{
							User:           "ubuntu",
							Host:           "192.168.0.1",
							PrivateKey:     "...rsa private key...",
							PrivateKeyPath: "/tmp/private.key",
							Passphrase:     "super secret passphrase",
							PassphrasePath: "/tmp/passphase",
						},
					},
					{
						Name: "default",
						SSH: &TransportSSH{
							User:           "ec2-user",
							Host:           "192.168.0.2",
							PrivateKey:     "...rsa private ec2 key...",
							PrivateKeyPath: "/tmp/ec2-private.key",
							Passphrase:     "ec2 super secret passphrase",
							PassphrasePath: "/tmp/ec2-passphase",
						},
					},
				},
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
				},
				Scenarios: []*Scenario{
					{
						Name: "default",
						Transport: &Transport{
							Name: "default",
							SSH: &TransportSSH{
								User:           "ec2-user",
								Host:           "192.168.0.2",
								PrivateKey:     "...rsa private ec2 key...",
								PrivateKeyPath: "/tmp/ec2-private.key",
								Passphrase:     "ec2 super secret passphrase",
								PassphrasePath: "/tmp/ec2-passphase",
							},
						},
						TerraformCLI: DefaultTerraformCLI(),
						Steps: []*ScenarioStep{
							{
								Name: "backend",
								Module: &Module{
									Name:   "backend",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"driver": cty.StringVal("postgres"),
									},
								},
							},
						},
					},
					{
						Name:         "ubuntu",
						TerraformCLI: DefaultTerraformCLI(),
						Transport: &Transport{
							Name: "ubuntu",
							SSH: &TransportSSH{
								User:           "ubuntu",
								Host:           "192.168.0.1",
								PrivateKey:     "...rsa private key...",
								PrivateKeyPath: "/tmp/private.key",
								Passphrase:     "super secret passphrase",
								PassphrasePath: "/tmp/passphase",
							},
						},
						Steps: []*ScenarioStep{
							{
								Name: "backend",
								Module: &Module{
									Name:   "backend",
									Source: modulePath,
									Attrs: map[string]cty.Value{
										"driver": cty.StringVal("postgres"),
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
			fp, diags := testDecodeHCL(t, []byte(test.hcl))
			if test.fail {
				require.True(t, diags.HasErrors(), diags.Error())
				return
			}
			require.False(t, diags.HasErrors(), diags.Error())
			testRequireEqualFP(t, fp, test.expected)
		})
	}
}
