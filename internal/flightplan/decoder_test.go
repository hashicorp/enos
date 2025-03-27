// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/net/context"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	hcl "github.com/hashicorp/hcl/v2"
)

// Test_Decode_FlightPlan tests decoding a flight plan.
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
				ScenarioBlocks: ScenarioBlocks{
					{
						Name: "basic",
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

func testDiagsToError(files map[string]*hcl.File, diags hcl.Diagnostics) error {
	if diags == nil || !diags.HasErrors() {
		return nil
	}
	msg := &strings.Builder{}
	writer := hcl.NewDiagnosticTextWriter(msg, files, 0, false)
	err := writer.WriteDiagnostics(diags)
	if err != nil {
		return fmt.Errorf("%w: %s", err, msg.String())
	}

	return errors.New(msg.String())
}

func testDecodeHCL(t *testing.T, hcl []byte, dt DecodeTarget, env ...string) (*FlightPlan, error) {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)
	decoder, err := NewDecoder(
		WithDecoderBaseDir(cwd),
		WithDecoderEnv(env),
		WithDecoderDecodeTarget(dt),
	)
	require.NoError(t, err)
	_, diags := decoder.FPParser.ParseHCL(hcl, "decoder-test.hcl")
	if diags != nil {
		require.False(t, diags.HasErrors(), testDiagsToError(decoder.ParserFiles(), diags))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fp, scenarioDecoder, moreDiags := decoder.Decode(ctx)
	diags = diags.Extend(moreDiags)
	if dt >= DecodeTargetScenariosNamesExpandVariants {
		diags = diags.Extend(scenarioDecoder.DecodeAll(ctx, fp))
	}

	return fp, testDiagsToError(decoder.ParserFiles(), diags)
}

type testCreateWireWorkspaceOpt func(*pb.Workspace)

func withTestCreateWireWorkspaceBody(body string) testCreateWireWorkspaceOpt {
	return func(ws *pb.Workspace) {
		if f := ws.GetFlightplan(); f == nil {
			ws.Flightplan = &pb.FlightPlan{}
		}

		ws.Flightplan.EnosHcl = map[string][]byte{
			"enos-test.hcl": []byte(body),
		}
	}
}

func testCreateWireWorkspace(t *testing.T, opts ...testCreateWireWorkspaceOpt) *pb.Workspace {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	ws := &pb.Workspace{
		Dir:    cwd,
		OutDir: t.TempDir(),
		Flightplan: &pb.FlightPlan{
			BaseDir: cwd,
		},
	}

	for i := range opts {
		opts[i](ws)
	}

	return ws
}

func testRequireEqualFP(t *testing.T, fp, expected *FlightPlan) {
	t.Helper()

	require.Len(t, fp.Modules, len(expected.Modules))
	require.Len(t, fp.ScenarioBlocks, len(expected.ScenarioBlocks))
	require.Len(t, fp.Providers, len(expected.Providers))

	if expected.Samples != nil {
		require.Len(t, fp.Samples, len(expected.Samples))
		for i := range expected.Samples {
			require.Equal(t, expected.Samples[i].Name, fp.Samples[i].Name)
			require.Equal(t, expected.Samples[i].Attributes, fp.Samples[i].Attributes)
			require.Len(t, expected.Samples[i].Subsets, len(fp.Samples[i].Subsets))
			for si := range expected.Samples[i].Subsets {
				require.Equal(t, expected.Samples[i].Subsets[si].Name, fp.Samples[i].Subsets[si].Name)
				require.Equal(t, expected.Samples[i].Subsets[si].ScenarioName, fp.Samples[i].Subsets[si].ScenarioName)
				require.Equal(t, expected.Samples[i].Subsets[si].ScenarioFilter, fp.Samples[i].Subsets[si].ScenarioFilter)
				require.Equal(t, expected.Samples[i].Subsets[si].Attributes, fp.Samples[i].Subsets[si].Attributes)
				if expected.Samples[i].Subsets[si].Matrix != nil {
					require.Truef(t,
						expected.Samples[i].Subsets[si].Matrix.EqualUnordered(fp.Samples[i].Subsets[si].Matrix),
						"expected equal unordered matrices: expected: \n%v\n, got: \n%v",
						expected.Samples[i].Subsets[si].Matrix, fp.Samples[i].Subsets[si].Matrix,
					)
				}
			}
		}
	}

	if expected.TerraformSettings != nil {
		require.Len(t, fp.TerraformSettings, len(expected.TerraformSettings))
		for i := range expected.TerraformSettings {
			require.Equal(t, expected.TerraformSettings[i], fp.TerraformSettings[i])
		}
	}

	if expected.Qualities != nil {
		require.Len(t, fp.Qualities, len(expected.Qualities))
		for i := range expected.Qualities {
			require.Equal(t, expected.Qualities[i], fp.Qualities[i])
		}
	}

	if expected.TerraformCLIs != nil {
		require.Len(t, fp.TerraformCLIs, len(expected.TerraformCLIs))
		for i := range expected.TerraformCLIs {
			require.Equal(t, expected.TerraformCLIs[i], fp.TerraformCLIs[i])
		}
	}

	for i := range expected.Modules {
		require.Equal(t, expected.Modules[i].Name, fp.Modules[i].Name)
		require.Equal(t, expected.Modules[i].Source, fp.Modules[i].Source)
		require.Equal(t, expected.Modules[i].Version, fp.Modules[i].Version)
		if len(expected.Modules[i].Attrs) > 0 {
			assert.Equal(t, expected.Modules[i].Attrs, fp.Modules[i].Attrs)
		}
	}

	if expected.ScenarioBlocks != nil {
		require.NotNil(t, fp.ScenarioBlocks)
	}
	for i := range expected.ScenarioBlocks {
		require.NotNil(t, fp.ScenarioBlocks)
		gotBlock := fp.ScenarioBlocks[i]
		require.NotNil(t, gotBlock)

		for j := range expected.ScenarioBlocks[i].Scenarios {
			require.Equal(t, expected.ScenarioBlocks[i].Name, gotBlock.Name)
			require.Equal(t, expected.ScenarioBlocks[i].Name, gotBlock.Name)
			require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Name, gotBlock.Scenarios[j].Name)
			require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Description, gotBlock.Scenarios[j].Description)
			require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].TerraformSetting, gotBlock.Scenarios[j].TerraformSetting)
			require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].TerraformCLI, gotBlock.Scenarios[j].TerraformCLI)
			if expected.ScenarioBlocks[i].Scenarios[j].Variants == nil {
				require.Nil(t, gotBlock.Scenarios[j].Variants)
			} else {
				expectedVariants := expected.ScenarioBlocks[i].Scenarios[j].Variants
				gotVariants := gotBlock.Scenarios[j].Variants
				require.NotNil(t, expectedVariants)
				require.NotNil(t, gotVariants)
				require.Equal(t, expectedVariants.Elements(), gotVariants.Elements())
			}
			require.Len(t, expected.ScenarioBlocks[i].Scenarios[j].Outputs, len(gotBlock.Scenarios[j].Outputs))

			if len(gotBlock.Scenarios[j].Outputs) > 0 {
				for oi := range expected.ScenarioBlocks[i].Scenarios[j].Outputs {
					require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Outputs[oi].Name, gotBlock.Scenarios[j].Outputs[oi].Name)
					require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Outputs[oi].Description, gotBlock.Scenarios[j].Outputs[oi].Description)
					require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Outputs[oi].Sensitive, gotBlock.Scenarios[j].Outputs[oi].Sensitive)
					eVal := expected.ScenarioBlocks[i].Scenarios[j].Outputs[oi].Value
					aVal := gotBlock.Scenarios[j].Outputs[oi].Value

					require.True(t, eVal.Type().Equals(aVal.Type()),
						"expected type %s, got %s", eVal.Type().FriendlyName(), aVal.Type().FriendlyName(),
					)
					if !eVal.IsNull() {
						testMostlyEqualStepVar(t, eVal, aVal)
					}
				}
			}

			require.Len(t, expected.ScenarioBlocks[i].Scenarios[j].Steps, len(gotBlock.Scenarios[j].Steps))
			for is := range expected.ScenarioBlocks[i].Scenarios[j].Steps {
				require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Steps[is].Name, gotBlock.Scenarios[j].Steps[is].Name)
				require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Steps[is].Description, gotBlock.Scenarios[j].Steps[is].Description)
				require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Steps[is].Providers, gotBlock.Scenarios[j].Steps[is].Providers)
				require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Steps[is].DependsOn, gotBlock.Scenarios[j].Steps[is].DependsOn)
				require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Steps[is].Verifies, gotBlock.Scenarios[j].Steps[is].Verifies)
				require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Steps[is].Skip, gotBlock.Scenarios[j].Steps[is].Skip)
				require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Steps[is].Module.Name, gotBlock.Scenarios[j].Steps[is].Module.Name)
				require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Steps[is].Module.Source, gotBlock.Scenarios[j].Steps[is].Module.Source)
				require.Equal(t, expected.ScenarioBlocks[i].Scenarios[j].Steps[is].Module.Version, gotBlock.Scenarios[j].Steps[is].Module.Version)

				for isa := range expected.ScenarioBlocks[i].Scenarios[j].Steps[is].Module.Attrs {
					eAttr := expected.ScenarioBlocks[i].Scenarios[j].Steps[is].Module.Attrs[isa]
					aAttr := fp.ScenarioBlocks[i].Scenarios[j].Steps[is].Module.Attrs[isa]

					require.Truef(
						t, eAttr.Type().Equals(aAttr.Type()),
						"expected %s type to have type %s, got %s", isa, eAttr.Type().GoString(), aAttr.Type().GoString(),
					)
					if !eAttr.IsNull() {
						testMostlyEqualStepVar(t, eAttr, aAttr)
					}
				}
			}
		}
	}

	for importName, provider := range expected.Providers {
		require.Equal(t, expected.Providers[importName], provider)
	}
}

// Scenario steps vars may have complicated values due to possibly embedded
// hcl.Traversal carrying lots of hcl.Range metadata and the like. Rather than
// trying to mock all of that data when testing so we can do true deep matching,
// we'll instead only check for attribute values that we care about.
func testMostlyEqualStepVar(t *testing.T, expected cty.Value, got cty.Value) {
	t.Helper()

	eVal, diags := StepVariableFromVal(expected)
	require.NotNil(t, eVal)
	require.False(t, diags.HasErrors(), diags.Error())
	aVal, diags := StepVariableFromVal(got)
	require.NotNil(t, aVal)
	require.False(t, diags.HasErrors(), diags.Error())
	require.Equal(t, eVal.Value, aVal.Value)
	require.Lenf(t, eVal.Traversal, len(aVal.Traversal),
		"expected %s to have a traversal of: %+v, got: %+v", eVal.Value.GoString(),
		eVal.Traversal, aVal.Traversal,
	)

	for i := range eVal.Traversal {
		if i == 0 {
			require.NotNil(t, aVal.Traversal)
			eTrav := eVal.Traversal[i]
			require.NotNil(t, eTrav)
			eAttr, ok := eTrav.(hcl.TraverseRoot)
			require.True(t, ok)
			aTrav := aVal.Traversal[i]
			require.NotNil(t, aTrav)
			aAttr, ok := aTrav.(hcl.TraverseRoot)
			require.True(t, ok)
			require.Equal(t, eAttr.Name, aAttr.Name)

			continue
		}
		eAttr, ok := eVal.Traversal[i].(hcl.TraverseAttr)
		require.True(t, ok)
		aAttr, ok := aVal.Traversal[i].(hcl.TraverseAttr)
		require.True(t, ok)
		require.Equal(t, eAttr.Name, aAttr.Name)
	}
}
