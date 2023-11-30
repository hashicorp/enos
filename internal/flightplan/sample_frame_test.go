package flightplan

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func Test_SampleFrame_Ref(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       *SampleFrame
		expected *pb.Ref_Sample
	}{
		"nil": {
			in:       nil,
			expected: nil,
		},
		"nil sample": {
			in:       &SampleFrame{},
			expected: nil,
		},
		"has sample": {
			in: &SampleFrame{
				Sample: &Sample{Name: "foo"},
			},
			expected: &pb.Ref_Sample{
				Id: &pb.Sample_ID{
					Name: "foo",
				},
			},
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.EqualValues(t, test.expected, test.in.Ref())
		})
	}
}

func Test_SampleFrame_Elements(t *testing.T) {
	t.Parallel()

	modulePath, err := filepath.Abs("./tests/simple_module")
	require.NoError(t, err)
	body := fmt.Sprintf(`
variable "input" {}

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

scenario "baz" {
  step "foo" {
    module = module.foo
  }
}

sample "all" {
  attributes = {
    aws-region = ["us-west-1", "us-west-2"] // Distribute these evenly between elements
    continue-on-error = false // Distribute to all elements
  }

  subset "merge" {
    scenario_name = "foo"

    attributes = {
      test-group = "merge"
    }

    matrix {
      length = ["fl1", "fl2"]
      width  = ["fw1", "fw2"]
    }
  }

  subset "override" {
    scenario_name = "bar"

    attributes = {
      test-group = "override"
      continue-on-error = true // Overridden attributes
      aws-region = ["eu-west-1", "us-east-1"]
    }

    matrix {
      length = ["bl1", "bl2"]
      width  = ["bw1", "bw2"]
    }
  }

  subset "nomatrix" {
    scenario_name = "baz"
  }
}`, modulePath)
	for subsetName, test := range map[string]struct {
		filter   *pb.Sample_Filter
		expected []*pb.Sample_Element
	}{
		"nomatrix": {
			filter: &pb.Sample_Filter{
				Subsets: []*pb.Sample_Subset_ID{
					{
						Name: "nomatrix",
					},
				},
			},
			expected: []*pb.Sample_Element{
				{
					Subset: &pb.Ref_Sample_Subset{
						Id: &pb.Sample_Subset_ID{
							Name: "nomatrix",
						},
					},
					Scenario: &pb.Ref_Scenario{
						Id: &pb.Scenario_ID{
							Name:     "baz",
							Filter:   "baz",
							Variants: nil,
							Uid:      "baa5a0964d3320fbc0c6a922140453c8513ea24ab8fd0577034804a967248096",
						},
					},
					Attributes: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"aws-region":        structpb.NewStringValue("us-west-1"),
							"continue-on-error": structpb.NewBoolValue(false),
						},
					},
				},
			},
		},
		"merge": {
			filter: &pb.Sample_Filter{
				Subsets: []*pb.Sample_Subset_ID{
					{
						Name: "merge",
					},
				},
			},
			expected: []*pb.Sample_Element{
				{
					Sample: nil,
					Subset: &pb.Ref_Sample_Subset{
						Id: &pb.Sample_Subset_ID{
							Name: "merge",
						},
					},
					Scenario: &pb.Ref_Scenario{
						Id: &pb.Scenario_ID{
							Name:   "foo",
							Filter: "foo length:fl1 width:fw1",
							Variants: &pb.Matrix_Vector{Elements: []*pb.Matrix_Element{
								{Key: "length", Value: "fl1"},
								{Key: "width", Value: "fw1"},
							}},
							Uid: "ed19801704ae6d375ac09ff073d79284b20e62f60d49763558bcd0916997e7a4",
						},
					},
					Attributes: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"aws-region":        structpb.NewStringValue("us-west-1"),
							"continue-on-error": structpb.NewBoolValue(false),
							"test-group":        structpb.NewStringValue("merge"),
						},
					},
				},
				{
					Subset: &pb.Ref_Sample_Subset{
						Id: &pb.Sample_Subset_ID{
							Name: "merge",
						},
					},
					Scenario: &pb.Ref_Scenario{
						Id: &pb.Scenario_ID{
							Name:   "foo",
							Filter: "foo length:fl1 width:fw2",
							Variants: &pb.Matrix_Vector{Elements: []*pb.Matrix_Element{
								{Key: "length", Value: "fl1"},
								{Key: "width", Value: "fw2"},
							}},
							Uid: "eb2c78bab08044b69415f9f3ba3efdb7a467e86af7012963c6b53b1bd847f708",
						},
					},
					Attributes: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"aws-region":        structpb.NewStringValue("us-west-2"),
							"continue-on-error": structpb.NewBoolValue(false),
							"test-group":        structpb.NewStringValue("merge"),
						},
					},
				},
				{
					Subset: &pb.Ref_Sample_Subset{
						Id: &pb.Sample_Subset_ID{
							Name: "merge",
						},
					},
					Scenario: &pb.Ref_Scenario{
						Id: &pb.Scenario_ID{
							Name:   "foo",
							Filter: "foo length:fl2 width:fw1",
							Variants: &pb.Matrix_Vector{Elements: []*pb.Matrix_Element{
								{Key: "length", Value: "fl2"},
								{Key: "width", Value: "fw1"},
							}},
							Uid: "5c36259d9aea446acb34e78a6633e8b3155febe562b8a4d813b724582d36f040",
						},
					},
					Attributes: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"aws-region":        structpb.NewStringValue("us-west-2"),
							"continue-on-error": structpb.NewBoolValue(false),
							"test-group":        structpb.NewStringValue("merge"),
						},
					},
				},
				{
					Subset: &pb.Ref_Sample_Subset{
						Id: &pb.Sample_Subset_ID{
							Name: "merge",
						},
					},
					Scenario: &pb.Ref_Scenario{
						Id: &pb.Scenario_ID{
							Name:   "foo",
							Filter: "foo length:fl2 width:fw2",
							Variants: &pb.Matrix_Vector{Elements: []*pb.Matrix_Element{
								{Key: "length", Value: "fl2"},
								{Key: "width", Value: "fw2"},
							}},
							Uid: "349a31acc232614b257b1ced5ac5ed5d393a4dc8f771ac97457d94f2587336e2",
						},
					},
					Attributes: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"aws-region":        structpb.NewStringValue("us-west-1"),
							"continue-on-error": structpb.NewBoolValue(false),
							"test-group":        structpb.NewStringValue("merge"),
						},
					},
				},
			},
		},
		"override": {
			filter: &pb.Sample_Filter{
				Subsets: []*pb.Sample_Subset_ID{
					{
						Name: "override",
					},
				},
			},
			expected: []*pb.Sample_Element{
				{
					Sample: nil,
					Subset: &pb.Ref_Sample_Subset{
						Id: &pb.Sample_Subset_ID{
							Name: "override",
						},
					},
					Scenario: &pb.Ref_Scenario{
						Id: &pb.Scenario_ID{
							Name:   "bar",
							Filter: "bar length:bl1 width:bw1",
							Variants: &pb.Matrix_Vector{Elements: []*pb.Matrix_Element{
								{Key: "length", Value: "bl1"},
								{Key: "width", Value: "bw1"},
							}},
							Uid: "de8d9dc11abad83db3fd15f12b8dce9b146ab0853bf48bcf16ded3997db7f0a0",
						},
					},
					Attributes: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"aws-region":        structpb.NewStringValue("eu-west-1"),
							"continue-on-error": structpb.NewBoolValue(true),
							"test-group":        structpb.NewStringValue("override"),
						},
					},
				},
				{
					Subset: &pb.Ref_Sample_Subset{
						Id: &pb.Sample_Subset_ID{
							Name: "override",
						},
					},
					Scenario: &pb.Ref_Scenario{
						Id: &pb.Scenario_ID{
							Name:   "bar",
							Filter: "bar length:bl1 width:bw2",
							Variants: &pb.Matrix_Vector{Elements: []*pb.Matrix_Element{
								{Key: "length", Value: "bl1"},
								{Key: "width", Value: "bw2"},
							}},
							Uid: "25b3607735cd66c02cd84aadffaa754a2ace079c5927b6f1c34ab65c76aacc95",
						},
					},
					Attributes: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"aws-region":        structpb.NewStringValue("us-east-1"),
							"continue-on-error": structpb.NewBoolValue(true),
							"test-group":        structpb.NewStringValue("override"),
						},
					},
				},
				{
					Subset: &pb.Ref_Sample_Subset{
						Id: &pb.Sample_Subset_ID{
							Name: "override",
						},
					},
					Scenario: &pb.Ref_Scenario{
						Id: &pb.Scenario_ID{
							Name:   "bar",
							Filter: "bar length:bl2 width:bw1",
							Variants: &pb.Matrix_Vector{Elements: []*pb.Matrix_Element{
								{Key: "length", Value: "bl2"},
								{Key: "width", Value: "bw1"},
							}},
							Uid: "89e14db1d1403490173b1667ebbf41f455944ba2ef300388fa4acd49dca562d6",
						},
					},
					Attributes: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"aws-region":        structpb.NewStringValue("us-east-1"),
							"continue-on-error": structpb.NewBoolValue(true),
							"test-group":        structpb.NewStringValue("override"),
						},
					},
				},
				{
					Subset: &pb.Ref_Sample_Subset{
						Id: &pb.Sample_Subset_ID{
							Name: "override",
						},
					},
					Scenario: &pb.Ref_Scenario{
						Id: &pb.Scenario_ID{
							Name:   "bar",
							Filter: "bar length:bl2 width:bw2",
							Variants: &pb.Matrix_Vector{Elements: []*pb.Matrix_Element{
								{Key: "length", Value: "bl2"},
								{Key: "width", Value: "bw2"},
							}},
							Uid: "b8f6d457be68608102b3603c16e5b96be64d3583ec5a18a6f34d95caae62c040",
						},
					},
					Attributes: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"aws-region":        structpb.NewStringValue("eu-west-1"),
							"continue-on-error": structpb.NewBoolValue(true),
							"test-group":        structpb.NewStringValue("override"),
						},
					},
				},
			},
		},
	} {
		subsetName := subsetName
		test := test
		t.Run(subsetName, func(t *testing.T) {
			t.Parallel()

			ws := testCreateWireWorkspace(t, withTestCreateWireWorkspaceBody(body))
			//nolint:gosec // we want deterministic randomness
			r := rand.New(rand.NewSource(1234)) // Use a static seed for deterministic attribute expansion
			fp, err := testDecodeHCL(t, ws.GetFlightplan().GetEnosHcl()["enos-test.hcl"], DecodeTargetAll)
			require.NoError(t, err)
			require.NotNil(t, fp)
			require.Len(t, fp.Samples, 1)
			samp := fp.Samples[0]

			frame, decRes := samp.Frame(context.Background(), ws, test.filter)
			require.Empty(t, decRes.GetDiagnostics())

			subFrame, ok := frame.SubsetFrames[subsetName]
			require.True(t, ok)

			elements, err := frame.Elements(subsetName, r, subFrame.Matrix)
			require.NoError(t, err)
			require.Len(t, elements, len(test.expected))

			for i := range test.expected {
				test.expected[i].Sample = samp.Ref()
				require.EqualValues(t, test.expected[i].GetSample(), elements[i].GetSample())
				require.EqualValues(t, test.expected[i].GetSubset(), elements[i].GetSubset())
				require.EqualValues(t, test.expected[i].GetScenario(), elements[i].GetScenario())
				gotAttrs := elements[i].GetAttributes().AsMap()
				for name, val := range test.expected[i].GetAttributes().AsMap() {
					attr, ok := gotAttrs[name]
					require.True(t, ok, "did not find expected attribute %s", name)
					require.EqualValues(t, val, attr)
				}
			}
		})
	}
}

func Test_SampleFrame_FilterMin(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in         *SampleFrame
		expected   int32
		shouldFail bool
	}{
		"nil": {
			in:         nil,
			shouldFail: true,
		},
		"no filter": {
			in:         &SampleFrame{},
			shouldFail: true,
		},
		"has min": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 2,
				},
			},
			expected: int32(2),
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			min, err := test.in.FilterMin()
			if test.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, min)
			}
		})
	}
}

func Test_SampleFrame_FilterMax(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in         *SampleFrame
		expected   int32
		shouldFail bool
	}{
		"nil": {
			in:         nil,
			shouldFail: true,
		},
		"no filter": {
			in:         &SampleFrame{},
			shouldFail: true,
		},
		"has max": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MaxElements: 16,
				},
			},
			expected: int32(16),
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			max, err := test.in.FilterMax()
			if test.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, max)
			}
		})
	}
}

func Test_SampleFrame_FilterPercentage(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in         *SampleFrame
		expected   float32
		shouldFail bool
	}{
		"nil": {
			in:         nil,
			shouldFail: true,
		},
		"no filter": {
			in:         &SampleFrame{},
			shouldFail: true,
		},
		"has pct": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					Percentage: 99,
				},
			},
			expected: float32(99),
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			pct, err := test.in.FilterPercentage()
			if test.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.InEpsilon(t, test.expected, pct, 0)
			}
		})
	}
}

func Test_SampleFrame_FilterMinMax(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in         *SampleFrame
		min        int32
		max        int32
		shouldFail bool
	}{
		"nil": {
			in:         nil,
			shouldFail: true,
		},
		"no filter": {
			in:         &SampleFrame{},
			shouldFail: true,
		},
		"no frames": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 0,
					MaxElements: 3,
				},
			},
			min: 0,
			max: 0,
		},
		"has pct unlimited max rounds": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: -1,
					Percentage:  99,
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			min: 1,
			max: 4,
		},
		"has pct and max": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: 3,
					Percentage:  99,
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			min: 1,
			max: 3,
		},
		"has pct and unlimited max": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: -1,
					Percentage:  75,
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			min: 1,
			max: 3,
		},
		"has pct less than 0": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: -1,
					Percentage:  -1,
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			min: 1,
			max: 4,
		},
		"has pct greater than 100": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: -1,
					Percentage:  200,
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			min: 1,
			max: 4,
		},
		"pct rounds down": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: 5,
					Percentage:  62, // 62% of 4 == 2.48 should round down to 2
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			min: 1,
			max: 2,
		},
		"pct rounds up": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: 5,
					Percentage:  63, // 63% of 4 == 2.52 should round up to 3
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			min: 1,
			max: 3,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			min, max, err := test.in.FilterMinMax()
			if test.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.min, min)
				require.Equal(t, test.max, max)
			}
		})
	}
}

func Test_SampleFrame_Size(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       *SampleFrame
		expected int32
	}{
		"nil": {
			in:       nil,
			expected: 0,
		},
		"no subset frames": {
			in:       &SampleFrame{},
			expected: 0,
		},
		"frames with no matrix": {
			in: &SampleFrame{
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						SampleSubset: &SampleSubset{
							Name: "foo",
						},
					},
					"foo_alias": {
						SampleSubset: &SampleSubset{
							Name:         "alias",
							ScenarioName: "foo",
						},
					},
				},
			},
			expected: 2,
		},
		"frames with matrix": {
			in: &SampleFrame{
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			expected: 4,
		},
		"mixed frames": {
			in: &SampleFrame{
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"baz": {
						SampleSubset: &SampleSubset{
							Name: "baz",
						},
					},
					"baz_alias": {
						SampleSubset: &SampleSubset{
							Name:         "alias",
							ScenarioName: "baz",
						},
					},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			expected: 6,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, test.in.Size())
		})
	}
}
