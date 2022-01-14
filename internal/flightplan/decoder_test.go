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

func testDecodeHCL(t *testing.T, hcl []byte) (*FlightPlan, hcl.Diagnostics) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	decoder, err := NewDecoder(WithDecoderBaseDir(cwd))
	require.NoError(t, err)
	diags := decoder.parseHCL(hcl, "decoder-test.hcl")
	require.False(t, diags.HasErrors(), diags.Error())
	return decoder.Decode()
}

func testRequireEqualFP(t *testing.T, fp, expected *FlightPlan) {
	require.Len(t, fp.Modules, len(expected.Modules))
	require.Len(t, fp.Scenarios, len(expected.Scenarios))

	if expected.TerraformCLIs != nil {
		require.Len(t, fp.TerraformCLIs, len(expected.TerraformCLIs))
		for i := range expected.TerraformCLIs {
			require.EqualValues(t, expected.TerraformCLIs[i], fp.TerraformCLIs[i])
		}
	}

	if expected.Transports != nil {
		require.Len(t, fp.Transports, len(expected.Transports))
		for i := range expected.Transports {
			require.EqualValues(t, expected.Transports[i], fp.Transports[i])
		}
	}

	for i := range expected.Modules {
		require.EqualValues(t, expected.Modules[i].Name, fp.Modules[i].Name)
		require.EqualValues(t, expected.Modules[i].Source, fp.Modules[i].Source)
		require.EqualValues(t, expected.Modules[i].Version, fp.Modules[i].Version)
		if len(expected.Modules[i].Attrs) > 0 {
			assert.EqualValues(t, expected.Modules[i].Attrs, fp.Modules[i].Attrs)
		}
	}

	for i := range expected.Scenarios {
		require.EqualValues(t, expected.Scenarios[i].Name, fp.Scenarios[i].Name)
		require.EqualValues(t, expected.Scenarios[i].Steps, fp.Scenarios[i].Steps)
		require.EqualValues(t, expected.Scenarios[i].Transport, fp.Scenarios[i].Transport)
		require.EqualValues(t, expected.Scenarios[i].TerraformCLI, fp.Scenarios[i].TerraformCLI)
	}
}

// Test_Decoder_parseDir tests loading enos configuration from a directory
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
		require.Equal(t, 0, len(decoder.Parser.Files()))
	})

	t.Run("two matching files", func(t *testing.T) {
		decoder := newDecoder("parse_dir_pass_two_matching_names")
		diags := decoder.Parse()
		require.False(t, diags.HasErrors())
		require.Equal(t, 2, len(decoder.Parser.Files()))
	})
}

// Test_Decode_FlightPlan tests decoding a flight plan
func Test_Decode_FlightPlan(t *testing.T) {
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
			desc: "minimal viable flight plan",
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
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				Modules: []*Module{
					{
						Name:   "backend",
						Source: modulePath,
					},
				},
				Scenarios: []*Scenario{
					{
						Name:         "basic",
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
					},
				},
			},
		},
		{
			desc: "invalid block",
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
			desc: "invalid attr",
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
