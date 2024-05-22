// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/enos/internal/diagnostics"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

func Test_SampleSubset_Frame(t *testing.T) {
	t.Parallel()

	modulePath, err := filepath.Abs("./tests/simple_module")
	require.NoError(t, err)

	for desc, test := range map[string]struct {
		ws       *pb.Workspace
		expected []*SampleSubsetFrame
	}{
		"complete": {
			ws: testCreateWireWorkspace(t, withTestCreateWireWorkspaceBody(fmt.Sprintf(`
module "foo" {
  source = "%s"
}

scenario "foo" {
  matrix {
    length = ["fl1", "fl2", "fl3"]
    width  = ["fw1", "fw2", "fw3"]
  }

  step "foo" {
    module = module.foo
  }
}

scenario "bar" {
  matrix {
    length = ["bl1", "bl2", "bl3"]
    width  = ["bw1", "bw2", "bw3"]
  }

  step "foo" {
    module = module.foo
  }
}

scenario "simple" {
  step "foo" {
    module = module.foo
  }
}

sample "foodle" {
  subset "foo" {
    matrix {
      length = ["fl2", "fl3"]
      width  = ["fw1", "fw3"]
    }

    attributes = {
      foo = "bar"
      hello = ["ohai", "howdy"]
    }
  }

  subset "barf" {
    scenario_filter = "bar length:bl1"
  }

  subset "simple" { }
}`, modulePath))),
			expected: []*SampleSubsetFrame{
				{
					SampleSubset: &SampleSubset{
						Name: "foo",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"foo":   cty.StringVal("bar"),
							"hello": cty.TupleVal([]cty.Value{cty.StringVal("ohai"), cty.StringVal("howdy")}),
						}),
					},
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(NewElement("length", "fl2"), NewElement("width", "fw1")),
						NewVector(NewElement("length", "fl2"), NewElement("width", "fw3")),
						NewVector(NewElement("length", "fl3"), NewElement("width", "fw1")),
						NewVector(NewElement("length", "fl3"), NewElement("width", "fw3")),
					}},
					ScenarioFilter: &pb.Scenario_Filter{
						Name: "foo",
					},
				},
				{
					SampleSubset: &SampleSubset{
						Name:           "barf",
						ScenarioFilter: "bar length:bl1",
					},
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(NewElement("length", "bl1"), NewElement("width", "bw1")),
						NewVector(NewElement("length", "bl1"), NewElement("width", "bw2")),
						NewVector(NewElement("length", "bl1"), NewElement("width", "bw3")),
					}},
					ScenarioFilter: &pb.Scenario_Filter{
						Name: "bar",
						Include: &pb.Matrix_Vector{
							Elements: []*pb.Matrix_Element{{Key: "length", Value: "bl1"}},
						},
					},
				},
				{
					SampleSubset: &SampleSubset{
						Name: "simple",
					},
					ScenarioFilter: &pb.Scenario_Filter{
						Name: "simple",
					},
				},
			},
		},
		"empty filter": {
			ws: testCreateWireWorkspace(t, withTestCreateWireWorkspaceBody(fmt.Sprintf(`
module "foo" {
  source = "%s"
}

scenario "foo" {
  matrix {
    length = ["fl1", "fl2", "fl3"]
    width  = ["fw1", "fw2", "fw3"]
  }

  step "foo" {
    module = module.foo
  }
}

sample "foo" {
  subset "foo" {
  }
}`, modulePath))),
			expected: []*SampleSubsetFrame{
				{
					SampleSubset: &SampleSubset{
						Name: "foo",
					},
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(NewElement("length", "fl1"), NewElement("width", "fw1")),
						NewVector(NewElement("length", "fl1"), NewElement("width", "fw2")),
						NewVector(NewElement("length", "fl1"), NewElement("width", "fw3")),
						NewVector(NewElement("length", "fl2"), NewElement("width", "fw1")),
						NewVector(NewElement("length", "fl2"), NewElement("width", "fw2")),
						NewVector(NewElement("length", "fl2"), NewElement("width", "fw3")),
						NewVector(NewElement("length", "fl3"), NewElement("width", "fw1")),
						NewVector(NewElement("length", "fl3"), NewElement("width", "fw2")),
						NewVector(NewElement("length", "fl3"), NewElement("width", "fw3")),
					}},
					ScenarioFilter: &pb.Scenario_Filter{
						Name: "foo",
					},
				},
			},
		},
		"no filter match name": {
			ws: testCreateWireWorkspace(t, withTestCreateWireWorkspaceBody(fmt.Sprintf(`
module "foo" {
  source = "%s"
}

scenario "foo" {
  matrix {
    length = ["fl1", "fl2", "fl3"]
    width  = ["fw1", "fw2", "fw3"]
  }

  step "foo" {
    module = module.foo
  }
}

sample "foo" {
  subset "bar" {
  }
}`, modulePath))),
			expected: nil,
		},

		"no filter match scenario_name": {
			ws: testCreateWireWorkspace(t, withTestCreateWireWorkspaceBody(fmt.Sprintf(`
module "foo" {
  source = "%s"
}

scenario "foo" {
  matrix {
    length = ["fl1", "fl2", "fl3"]
    width  = ["fw1", "fw2", "fw3"]
  }

  step "foo" {
    module = module.foo
  }
}

sample "foo" {
  subset "foo" {
    scenario_name = "bar"
  }
}`, modulePath))),
			expected: nil,
		},
		"no filter match scenario_filter": {
			ws: testCreateWireWorkspace(t, withTestCreateWireWorkspaceBody(fmt.Sprintf(`
module "foo" {
  source = "%s"
}

scenario "foo" {
  matrix {
    length = ["fl1", "fl2", "fl3"]
    width  = ["fw1", "fw2", "fw3"]
  }

  step "foo" {
    module = module.foo
  }
}

sample "foo" {
  subset "foo" {
    scenario_name = "bar"
  }
}`, modulePath))),
			expected: nil,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			fp, err := testDecodeHCL(t, test.ws.GetFlightplan().GetEnosHcl()["enos-test.hcl"], DecodeTargetAll)
			require.NoError(t, err)
			require.NotNil(t, fp)
			require.GreaterOrEqual(t, 1, len(fp.Samples))
			samp := fp.Samples[0]

			// Handle cases where we don't expect to get a valid frame
			if test.expected == nil {
				for i := range samp.Subsets {
					frame, decRes := samp.Subsets[i].Frame(context.Background(), test.ws)
					require.NotEmpty(t, decRes.GetDiagnostics())
					testRequireEqualSampleSubsetFrame(t, nil, frame)
				}

				return
			}

			// Make sure all of our frames match
			require.Len(t, test.expected, len(samp.Subsets))
			for i := range test.expected {
				sub := samp.Subsets[i]
				frame, decRes := sub.Frame(context.Background(), test.ws)
				msg := "expected an equal frame"
				for _, d := range decRes.GetDiagnostics() {
					msg += " " + diagnostics.String(d)
				}
				require.Emptyf(t, decRes.GetDiagnostics(), msg)

				testRequireEqualSampleSubsetFrame(t, test.expected[i], frame)
			}
		})
	}
}

func testRequireEqualSampleSubsetFrame(t *testing.T, expected, got *SampleSubsetFrame) {
	t.Helper()

	if expected == nil {
		require.Nil(t, got)

		return
	}

	require.EqualValues(t, expected.SampleSubset.SampleName, got.SampleSubset.SampleName)
	require.EqualValues(t, expected.SampleSubset.Name, got.SampleSubset.Name)
	require.EqualValues(t, expected.SampleSubset.ScenarioName, got.SampleSubset.ScenarioName)
	require.EqualValues(t, expected.SampleSubset.ScenarioFilter, got.SampleSubset.ScenarioFilter)
	require.EqualValues(t, expected.SampleSubset.Attributes, got.SampleSubset.Attributes)
	require.EqualValues(t, expected.ScenarioFilter.GetName(), got.ScenarioFilter.GetName())
	require.EqualValues(t, expected.ScenarioFilter.GetExclude(), got.ScenarioFilter.GetExclude())
	require.EqualValues(t, expected.ScenarioFilter.GetInclude(), got.ScenarioFilter.GetInclude())
	require.EqualValues(t, expected.ScenarioFilter.GetSelectAll(), got.ScenarioFilter.GetSelectAll())
	require.Truef(t, expected.Matrix.EqualUnordered(got.Matrix), fmt.Sprintf(
		"expected matrix vectors: \n%s\ngot matrix vectors: \n%s\ndifference: \n%s\n",
		expected.Matrix.String(),
		got.Matrix.String(),
		expected.Matrix.SymmetricDifferenceUnordered(got.Matrix).String(),
	))
}

func Test_SampleSubsetFrame_Size(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       func() *SampleSubsetFrame
		expected int32
	}{
		"nil": {
			in:       func() *SampleSubsetFrame { return nil },
			expected: 0,
		},
		"no subset and no matrix": {
			in:       func() *SampleSubsetFrame { return &SampleSubsetFrame{} },
			expected: 0,
		},
		"no matrices": {
			in: func() *SampleSubsetFrame {
				return &SampleSubsetFrame{
					SampleSubset: &SampleSubset{
						SampleName: "my_sample",
						Name:       "smoke",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"foo":   cty.StringVal("bar"),
							"hello": cty.TupleVal([]cty.Value{cty.StringVal("ohai"), cty.StringVal("howdy")}),
						}),
					},
				}
			},
			expected: 1,
		},
		"subset frame matrix": {
			in: func() *SampleSubsetFrame {
				return &SampleSubsetFrame{
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
					}},
				}
			},
			expected: 2,
		},
		"subset has matrix but subset frame has no matrix": {
			in: func() *SampleSubsetFrame {
				return &SampleSubsetFrame{
					SampleSubset: &SampleSubset{
						SampleName: "my_sample",
						Name:       "smoke",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"foo":   cty.StringVal("bar"),
							"hello": cty.TupleVal([]cty.Value{cty.StringVal("ohai"), cty.StringVal("howdy")}),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
						}},
					},
				}
			},
			expected: 0,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, test.in().Size())
		})
	}
}

func Test_SampleSubsetFrame_Validate(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in         func() *SampleSubsetFrame
		shouldFail bool
	}{
		"nil": {
			in:         func() *SampleSubsetFrame { return nil },
			shouldFail: true,
		},
		"no subset and no matrix": {
			in:         func() *SampleSubsetFrame { return &SampleSubsetFrame{} },
			shouldFail: true,
		},
		"no matrices": {
			in: func() *SampleSubsetFrame {
				return &SampleSubsetFrame{
					SampleSubset: &SampleSubset{
						SampleName: "my_sample",
						Name:       "smoke",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"foo":   cty.StringVal("bar"),
							"hello": cty.TupleVal([]cty.Value{cty.StringVal("ohai"), cty.StringVal("howdy")}),
						}),
					},
				}
			},
		},
		"subset frame matrix": {
			in: func() *SampleSubsetFrame {
				return &SampleSubsetFrame{
					SampleSubset: &SampleSubset{
						SampleName: "my_sample",
						Name:       "smoke",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"foo":   cty.StringVal("bar"),
							"hello": cty.TupleVal([]cty.Value{cty.StringVal("ohai"), cty.StringVal("howdy")}),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
						}},
					},
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
					}},
				}
			},
		},
		"subset has matrix but subset frame has no matrix": {
			in: func() *SampleSubsetFrame {
				return &SampleSubsetFrame{
					SampleSubset: &SampleSubset{
						SampleName: "my_sample",
						Name:       "smoke",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"foo":   cty.StringVal("bar"),
							"hello": cty.TupleVal([]cty.Value{cty.StringVal("ohai"), cty.StringVal("howdy")}),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
						}},
					},
				}
			},
			shouldFail: true,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			if test.shouldFail {
				require.Error(t, test.in().Validate())
			} else {
				require.NoError(t, test.in().Validate())
			}
		})
	}
}
