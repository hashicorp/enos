package flightplan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
)

func testDiagsToError(files map[string]*hcl.File, diags hcl.Diagnostics) error {
	if !diags.HasErrors() {
		return nil
	}
	msg := &strings.Builder{}
	writer := hcl.NewDiagnosticTextWriter(msg, files, 0, false)
	err := writer.WriteDiagnostics(diags)
	if err != nil {
		return fmt.Errorf("%w: %s", err, msg.String())
	}

	return fmt.Errorf(msg.String())
}

func testDecodeHCL(t *testing.T, hcl []byte) (*FlightPlan, error) {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	decoder, err := NewDecoder(WithDecoderBaseDir(cwd))
	require.NoError(t, err)
	diags := decoder.parseHCL(hcl, "decoder-test.hcl")
	require.False(t, diags.HasErrors(), testDiagsToError(decoder.Parser.Files(), diags))
	fp, diags := decoder.Decode()
	return fp, testDiagsToError(decoder.Parser.Files(), diags)
}

func testRequireEqualFP(t *testing.T, fp, expected *FlightPlan) {
	t.Helper()
	require.Len(t, fp.Modules, len(expected.Modules))
	require.Len(t, fp.Scenarios, len(expected.Scenarios))
	require.Len(t, fp.Providers, len(expected.Providers))

	if expected.TerraformSettings != nil {
		require.Len(t, fp.TerraformSettings, len(expected.TerraformSettings))
		for i := range expected.TerraformSettings {
			require.EqualValues(t, expected.TerraformSettings[i], fp.TerraformSettings[i])
		}
	}

	if expected.TerraformCLIs != nil {
		require.Len(t, fp.TerraformCLIs, len(expected.TerraformCLIs))
		for i := range expected.TerraformCLIs {
			require.EqualValues(t, expected.TerraformCLIs[i], fp.TerraformCLIs[i])
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
		require.EqualValues(t, expected.Scenarios[i].TerraformSetting, fp.Scenarios[i].TerraformSetting)
		require.EqualValues(t, expected.Scenarios[i].TerraformCLI, fp.Scenarios[i].TerraformCLI)

		require.Len(t, expected.Scenarios[i].Steps, len(fp.Scenarios[i].Steps))
		for is := range expected.Scenarios[i].Steps {
			require.EqualValues(t, expected.Scenarios[i].Steps[is].Name, fp.Scenarios[i].Steps[is].Name)
			require.EqualValues(t, expected.Scenarios[i].Steps[is].Providers, fp.Scenarios[i].Steps[is].Providers)
			require.EqualValues(t, expected.Scenarios[i].Steps[is].Module.Name, fp.Scenarios[i].Steps[is].Module.Name)
			require.EqualValues(t, expected.Scenarios[i].Steps[is].Module.Source, fp.Scenarios[i].Steps[is].Module.Source)
			require.EqualValues(t, expected.Scenarios[i].Steps[is].Module.Version, fp.Scenarios[i].Steps[is].Module.Version)

			for isa := range expected.Scenarios[i].Steps[is].Module.Attrs {
				eAttr := expected.Scenarios[i].Steps[is].Module.Attrs[isa]
				aAttr := fp.Scenarios[i].Steps[is].Module.Attrs[isa]

				require.True(t, eAttr.Type().Equals(aAttr.Type()))
				if !eAttr.IsNull() {
					testMostlyEqualStepVar(t, eAttr, aAttr)
				}
			}
		}
	}

	for importName, provider := range expected.Providers {
		require.EqualValues(t, expected.Providers[importName], provider)
	}
}

// Scenario steps vars may have complicated values due to possibly embedded
// hcl.Traversal carrying lots of hcl.Range metadata and the like. Rather than
// trying to mock all of that data when testing so we can do true deep matching,
// we'll instead only check for attribute values that we care about.
func testMostlyEqualStepVar(t *testing.T, expected cty.Value, got cty.Value) {
	t.Helper()

	eVal, diags := StepVariableFromVal(expected)
	require.False(t, diags.HasErrors(), diags.Error())
	aVal, diags := StepVariableFromVal(got)
	require.False(t, diags.HasErrors(), diags.Error())
	require.EqualValues(t, eVal.Value, aVal.Value)
	require.Len(t, eVal.Traversal, len(aVal.Traversal))
	for i := range eVal.Traversal {
		if i == 0 {
			eAttr, ok := eVal.Traversal[i].(hcl.TraverseRoot)
			require.True(t, ok)
			aAttr, ok := aVal.Traversal[i].(hcl.TraverseRoot)
			require.True(t, ok)
			require.EqualValues(t, eAttr.Name, aAttr.Name)
			continue
		}
		eAttr, ok := eVal.Traversal[i].(hcl.TraverseAttr)
		require.True(t, ok)
		aAttr, ok := aVal.Traversal[i].(hcl.TraverseAttr)
		require.True(t, ok)
		require.EqualValues(t, eAttr.Name, aAttr.Name)
	}
}

// Test_Decoder_parseDir tests loading enos configuration from a directory
func Test_Decoder_parseDir(t *testing.T) {
	t.Helper()
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
	t.Helper()
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
