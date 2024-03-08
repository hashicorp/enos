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

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// Test_Sample_Decode tests decoding the "sample" block of the DSL.
func Test_Sample_Decode(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		body     string
		env      []string
		expected *FlightPlan
		fail     bool
	}{
		"minimal config": {
			body: `
sample "foo" {
  subset "bar" { }
}`,
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				Samples: []*Sample{
					{
						Name: "foo",
						Subsets: []*SampleSubset{
							{
								Name: "bar",
							},
						},
					},
				},
			},
		},
		"maximal config": {
			body: `
sample "valid_name" {
  attributes = {
    aws-region = ["us-west-1", "us-west-2"] // Distribute these evenly between elements
    continue-on-error = false // Distribute to all elements
  }

  // More than one subset for a scenario
  subset "replication_consul" {
    scenario_name = "replication"

    attributes = {
      continue-on-error = true // Overridden attributes
    }

    matrix {
      arch = ["amd64"]
      artifact_source = ["artifactory"]
      artifact_type = ["package"]
      distro = ["rhel", "ubuntu"]
      edition = ["ent.hsm"]
      primary_backend = ["consul"]

      include { }
      exclude { }
    }
  }

  subset "replication_raft" {
    scenario_name = "replication"
    attributes = var.input // attributes from a variable or global

    matrix {
      arch = ["amd64"]
      artifact_source = ["artifactory"]
      artifact_type = ["bundle"]
      distro = ["rhel", "centos", "amz"]
      edition = ["ent.hsm"]
      primary_backend = ["raft"]
    }
  }

  subset "smoke" {
    matrix {
      arch = ["amd64"]
      artifact_source = ["artifactory"]
      artifact_type = ["package"]
      distro = ["rhel", "centos", "amz"]
      edition = ["ent.hsm"]
    }
  }

  subset "alias" {
    scenario_filter = "upgrade arch:amd64 artifact_source:artifactory artifact_type:package distro:rhel edition:ent"
  }
}

variable "input" {}
`,
			env: []string{`ENOS_VAR_input={something="thing",another_thing="another"}`},
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				Samples: []*Sample{
					{
						Name: "valid_name",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"aws-region":        cty.TupleVal([]cty.Value{cty.StringVal("us-west-1"), cty.StringVal("us-west-2")}),
							"continue-on-error": cty.BoolVal(false),
						}),
						Subsets: []*SampleSubset{
							{
								Name:         "replication_consul",
								ScenarioName: "replication",
								Attributes: cty.ObjectVal(map[string]cty.Value{
									"continue-on-error": cty.BoolVal(true),
								}),
								Matrix: &Matrix{Vectors: []*Vector{
									NewVector(
										NewElement("arch", "amd64"),
										NewElement("artifact_source", "artifactory"),
										NewElement("artifact_type", "package"),
										NewElement("distro", "rhel"),
										NewElement("edition", "ent.hsm"),
										NewElement("primary_backend", "consul"),
									),
									NewVector(
										NewElement("arch", "amd64"),
										NewElement("artifact_source", "artifactory"),
										NewElement("artifact_type", "package"),
										NewElement("distro", "ubuntu"),
										NewElement("edition", "ent.hsm"),
										NewElement("primary_backend", "consul"),
									),
								}},
							},
							{
								Name:         "replication_raft",
								ScenarioName: "replication",
								Attributes: cty.ObjectVal(map[string]cty.Value{
									"something":     cty.StringVal("thing"),
									"another_thing": cty.StringVal("another"),
								}),
								Matrix: &Matrix{Vectors: []*Vector{
									NewVector(
										NewElement("arch", "amd64"),
										NewElement("artifact_source", "artifactory"),
										NewElement("artifact_type", "bundle"),
										NewElement("distro", "rhel"),
										NewElement("edition", "ent.hsm"),
										NewElement("primary_backend", "raft"),
									),
									NewVector(
										NewElement("arch", "amd64"),
										NewElement("artifact_source", "artifactory"),
										NewElement("artifact_type", "bundle"),
										NewElement("distro", "centos"),
										NewElement("edition", "ent.hsm"),
										NewElement("primary_backend", "raft"),
									),
									NewVector(
										NewElement("arch", "amd64"),
										NewElement("artifact_source", "artifactory"),
										NewElement("artifact_type", "bundle"),
										NewElement("distro", "amz"),
										NewElement("edition", "ent.hsm"),
										NewElement("primary_backend", "raft"),
									),
								}},
							},
							{
								Name: "smoke",
								Matrix: &Matrix{Vectors: []*Vector{
									NewVector(
										NewElement("arch", "amd64"),
										NewElement("artifact_source", "artifactory"),
										NewElement("artifact_type", "package"),
										NewElement("distro", "rhel"),
										NewElement("edition", "ent.hsm"),
									),
									NewVector(
										NewElement("arch", "amd64"),
										NewElement("artifact_source", "artifactory"),
										NewElement("artifact_type", "package"),
										NewElement("distro", "centos"),
										NewElement("edition", "ent.hsm"),
									),
									NewVector(
										NewElement("arch", "amd64"),
										NewElement("artifact_source", "artifactory"),
										NewElement("artifact_type", "package"),
										NewElement("distro", "amz"),
										NewElement("edition", "ent.hsm"),
									),
								}},
							},
							{
								Name:           "alias",
								ScenarioFilter: "upgrade arch:amd64 artifact_source:artifactory artifact_type:package distro:rhel edition:ent",
							},
						},
					},
				},
			},
		},
		"invalid identifier": {
			body: `
sample "foo:" {
  subset "bar" { }
}
`,
			fail: true,
		},
		"unknown block": {
			body: `
sample "foo" {
  subset "bar" { }
  not_a_supported_block { }
}
`,
			fail: true,
		},
		"unknown attr": {
			body: `
sample "foo" {
  subset "bar" { }
  not_an_attr = "something"
}
`,
			fail: true,
		},
		"invalid attributes value": {
			body: `
sample "foo" {
  attributes = "not_a_map"
  subset "bar" { }
}
`,
			fail: true,
		},
		"no subsets": {
			body: `sample "foo" { }`,
			fail: true,
		},
		"invalid subset identifier": {
			body: `
sample "foo" {
  subset "bar-" { }
}
`,
			fail: true,
		},
		"unknown subset block": {
			body: `
sample "foo" {
  subset "bar" {
    not_a_supported_block { }
  }
}
`,
			fail: true,
		},
		"unknown subset attr": {
			body: `
sample "foo" {
  subset "bar" {
    not_an_att = "something"
  }
}
`,
			fail: true,
		},
		"invalid subset attributes value": {
			body: `
sample "foo" {
  subset "bar" {
    attributes = false
  }
}
`,
			fail: true,
		},
		"invalid subset scenario_name value": {
			body: `
sample "foo" {
  subset "bar" {
    scenario_name = "!-"
  }
}
`,
			fail: true,
		},
		"invalid subset scenario_filter value": {
			body: `
sample "foo" {
  subset "bar" {
    scenario_filter = ["not a string"]
  }
}
`,
			fail: true,
		},
		"scenario_name and scenario_filter both defined": {
			body: `
sample "foo" {
  subset "bar" {
    scenario_name = "upgrade"
    scenario_filter = "smoke backend:raft"
  }
}
`,
			fail: true,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			fp, err := testDecodeHCL(t, []byte(test.body), DecodeTargetAll, test.env...)
			if test.fail {
				require.Error(t, err)

				return
			}
			require.NoError(t, err)
			testRequireEqualFP(t, fp, test.expected)
		})
	}
}

func Test_Sample_Frame(t *testing.T) {
	t.Parallel()

	modulePath, err := filepath.Abs("./tests/simple_module")
	require.NoError(t, err)

	for desc, test := range map[string]struct {
		ws       *pb.Workspace
		filter   *pb.Sample_Filter
		expected *SampleFrame
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
			expected: &SampleFrame{
				SubsetFrames: map[string]*SampleSubsetFrame{
					"foo": {
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
					"barf": {
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
					"simple": {
						SampleSubset: &SampleSubset{
							Name: "simple",
						},
						ScenarioFilter: &pb.Scenario_Filter{
							Name: "simple",
						},
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
			expected: &SampleFrame{
				SubsetFrames: map[string]*SampleSubsetFrame{
					"foo": {
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
		},
		"filter sample by name with include": {
			filter: &pb.Sample_Filter{
				Sample: &pb.Ref_Sample{
					Id: &pb.Sample_ID{
						Name: "foodle",
					},
				},
				Subsets: []*pb.Sample_Subset_ID{
					{Name: "foo"},
				},
			},
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
  }

  subset "bar" {
    scenario_filter = "foo length:fl1"
  }

  subset "simple" { }
}`, modulePath))),
			expected: &SampleFrame{
				SubsetFrames: map[string]*SampleSubsetFrame{
					"foo": {
						SampleSubset: &SampleSubset{
							Name: "foo",
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
				},
			},
		},
		"filter sample by name with exclude": {
			filter: &pb.Sample_Filter{
				Sample: &pb.Ref_Sample{
					Id: &pb.Sample_ID{
						Name: "foodle",
					},
				},
				ExcludeSubsets: []*pb.Sample_Subset_ID{
					{Name: "foo"},
				},
			},
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
  }

  subset "bar" {
    scenario_filter = "foo length:fl1"
  }

  subset "simple" { }
}`, modulePath))),
			expected: &SampleFrame{
				SubsetFrames: map[string]*SampleSubsetFrame{
					"bar": {
						SampleSubset: &SampleSubset{
							Name:           "bar",
							ScenarioFilter: "bar length:bl1",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("length", "fl1"), NewElement("width", "fw1")),
							NewVector(NewElement("length", "fl1"), NewElement("width", "fw2")),
							NewVector(NewElement("length", "fl1"), NewElement("width", "fw3")),
						}},
						ScenarioFilter: &pb.Scenario_Filter{
							Name: "bar",
							Include: &pb.Matrix_Vector{
								Elements: []*pb.Matrix_Element{{Key: "length", Value: "bl1"}},
							},
						},
					},
					"simple": {
						SampleSubset: &SampleSubset{
							Name: "simple",
						},
						ScenarioFilter: &pb.Scenario_Filter{
							Name: "simple",
						},
					},
				},
			},
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			fp, err := testDecodeHCL(t, test.ws.GetFlightplan().GetEnosHcl()["enos-test.hcl"], DecodeTargetAll)
			require.NoError(t, err)
			require.NotNil(t, fp)
			require.Len(t, fp.Samples, 1)
			samp := fp.Samples[0]
			frame, decRes := samp.Frame(context.Background(), test.ws, test.filter)
			require.EqualValues(t, samp, frame.Sample)

			// Handle cases where we don't expect to get a valid frame
			if test.expected == nil {
				require.Equal(t, int32(0), frame.Size())
				require.Empty(t, decRes.GetDiagnostics())

				return
			}

			// Make sure all of our frames match
			require.Len(t, test.expected.SubsetFrames, len(frame.SubsetFrames))
			for k, gotFrame := range test.expected.SubsetFrames {
				expFrame := test.expected.SubsetFrames[k]

				testRequireEqualSampleSubsetFrame(t, expFrame, gotFrame)
			}
		})
	}
}

// Test_Sample_filterSubsets.
func Test_Sample_filterSubsets(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       *Sample
		filter   *pb.Sample_Filter
		expected []*SampleSubset
	}{
		"nil sample": {
			in:       nil,
			filter:   nil,
			expected: nil,
		},
		"no filter": {
			in: &Sample{
				Name: "valid_name",
				Attributes: cty.ObjectVal(map[string]cty.Value{
					"aws-region":        cty.TupleVal([]cty.Value{cty.StringVal("us-west-1"), cty.StringVal("us-west-2")}),
					"continue-on-error": cty.BoolVal(false),
				}),
				Subsets: []*SampleSubset{
					{
						Name:         "replication_consul",
						ScenarioName: "replication",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"continue-on-error": cty.BoolVal(true),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "consul"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "ubuntu"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "consul"),
							),
						}},
					},
					{
						Name:         "replication_raft",
						ScenarioName: "replication",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"something":     cty.StringVal("thing"),
							"another_thing": cty.StringVal("another"),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "centos"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "amz"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
						}},
					},
					{
						Name: "smoke",
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "centos"),
								NewElement("edition", "ent.hsm"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "amz"),
								NewElement("edition", "ent.hsm"),
							),
						}},
					},
					{
						Name:           "alias",
						ScenarioFilter: "upgrade arch:amd64 artifact_source:artifactory artifact_type:package distro:rhel edition:ent",
					},
				},
			},
			filter: nil,
			expected: []*SampleSubset{
				{
					Name:         "replication_consul",
					ScenarioName: "replication",
					Attributes: cty.ObjectVal(map[string]cty.Value{
						"continue-on-error": cty.BoolVal(true),
					}),
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "rhel"),
							NewElement("edition", "ent.hsm"),
							NewElement("primary_backend", "consul"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "ubuntu"),
							NewElement("edition", "ent.hsm"),
							NewElement("primary_backend", "consul"),
						),
					}},
				},
				{
					Name:         "replication_raft",
					ScenarioName: "replication",
					Attributes: cty.ObjectVal(map[string]cty.Value{
						"something":     cty.StringVal("thing"),
						"another_thing": cty.StringVal("another"),
					}),
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "bundle"),
							NewElement("distro", "rhel"),
							NewElement("edition", "ent.hsm"),
							NewElement("primary_backend", "raft"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "bundle"),
							NewElement("distro", "centos"),
							NewElement("edition", "ent.hsm"),
							NewElement("primary_backend", "raft"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "bundle"),
							NewElement("distro", "amz"),
							NewElement("edition", "ent.hsm"),
							NewElement("primary_backend", "raft"),
						),
					}},
				},
				{
					Name: "smoke",
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "rhel"),
							NewElement("edition", "ent.hsm"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "centos"),
							NewElement("edition", "ent.hsm"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "amz"),
							NewElement("edition", "ent.hsm"),
						),
					}},
				},
				{
					Name:           "alias",
					ScenarioFilter: "upgrade arch:amd64 artifact_source:artifactory artifact_type:package distro:rhel edition:ent",
				},
			},
		},
		"includes": {
			in: &Sample{
				Name: "valid_name",
				Attributes: cty.ObjectVal(map[string]cty.Value{
					"aws-region":        cty.TupleVal([]cty.Value{cty.StringVal("us-west-1"), cty.StringVal("us-west-2")}),
					"continue-on-error": cty.BoolVal(false),
				}),
				Subsets: []*SampleSubset{
					{
						Name:         "replication_consul",
						ScenarioName: "replication",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"continue-on-error": cty.BoolVal(true),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "consul"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "ubuntu"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "consul"),
							),
						}},
					},
					{
						Name:         "replication_raft",
						ScenarioName: "replication",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"something":     cty.StringVal("thing"),
							"another_thing": cty.StringVal("another"),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "centos"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "amz"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
						}},
					},
					{
						Name: "smoke",
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "centos"),
								NewElement("edition", "ent.hsm"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "amz"),
								NewElement("edition", "ent.hsm"),
							),
						}},
					},
					{
						Name:           "alias",
						ScenarioFilter: "upgrade arch:amd64 artifact_source:artifactory artifact_type:package distro:rhel edition:ent",
					},
				},
			},
			filter: &pb.Sample_Filter{
				Subsets: []*pb.Sample_Subset_ID{
					{
						Name: "replication_consul",
					},
					{
						Name: "alias",
					},
				},
			},
			expected: []*SampleSubset{
				{
					Name:         "replication_consul",
					ScenarioName: "replication",
					Attributes: cty.ObjectVal(map[string]cty.Value{
						"continue-on-error": cty.BoolVal(true),
					}),
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "rhel"),
							NewElement("edition", "ent.hsm"),
							NewElement("primary_backend", "consul"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "ubuntu"),
							NewElement("edition", "ent.hsm"),
							NewElement("primary_backend", "consul"),
						),
					}},
				},
				{
					Name:           "alias",
					ScenarioFilter: "upgrade arch:amd64 artifact_source:artifactory artifact_type:package distro:rhel edition:ent",
				},
			},
		},
		"excludes": {
			in: &Sample{
				Name: "valid_name",
				Attributes: cty.ObjectVal(map[string]cty.Value{
					"aws-region":        cty.TupleVal([]cty.Value{cty.StringVal("us-west-1"), cty.StringVal("us-west-2")}),
					"continue-on-error": cty.BoolVal(false),
				}),
				Subsets: []*SampleSubset{
					{
						Name:         "replication_consul",
						ScenarioName: "replication",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"continue-on-error": cty.BoolVal(true),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "consul"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "ubuntu"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "consul"),
							),
						}},
					},
					{
						Name:         "replication_raft",
						ScenarioName: "replication",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"something":     cty.StringVal("thing"),
							"another_thing": cty.StringVal("another"),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "centos"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "amz"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
						}},
					},
					{
						Name: "smoke",
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "centos"),
								NewElement("edition", "ent.hsm"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "amz"),
								NewElement("edition", "ent.hsm"),
							),
						}},
					},
					{
						Name:           "alias",
						ScenarioFilter: "upgrade arch:amd64 artifact_source:artifactory artifact_type:package distro:rhel edition:ent",
					},
				},
			},
			filter: &pb.Sample_Filter{
				ExcludeSubsets: []*pb.Sample_Subset_ID{
					{
						Name: "replication_consul",
					},
					{
						Name: "alias",
					},
				},
			},
			expected: []*SampleSubset{
				{
					Name:         "replication_raft",
					ScenarioName: "replication",
					Attributes: cty.ObjectVal(map[string]cty.Value{
						"something":     cty.StringVal("thing"),
						"another_thing": cty.StringVal("another"),
					}),
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "bundle"),
							NewElement("distro", "rhel"),
							NewElement("edition", "ent.hsm"),
							NewElement("primary_backend", "raft"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "bundle"),
							NewElement("distro", "centos"),
							NewElement("edition", "ent.hsm"),
							NewElement("primary_backend", "raft"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "bundle"),
							NewElement("distro", "amz"),
							NewElement("edition", "ent.hsm"),
							NewElement("primary_backend", "raft"),
						),
					}},
				},
				{
					Name: "smoke",
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "rhel"),
							NewElement("edition", "ent.hsm"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "centos"),
							NewElement("edition", "ent.hsm"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "amz"),
							NewElement("edition", "ent.hsm"),
						),
					}},
				},
			},
		},
		"include and excludes": {
			in: &Sample{
				Name: "valid_name",
				Attributes: cty.ObjectVal(map[string]cty.Value{
					"aws-region":        cty.TupleVal([]cty.Value{cty.StringVal("us-west-1"), cty.StringVal("us-west-2")}),
					"continue-on-error": cty.BoolVal(false),
				}),
				Subsets: []*SampleSubset{
					{
						Name:         "replication_consul",
						ScenarioName: "replication",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"continue-on-error": cty.BoolVal(true),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "consul"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "ubuntu"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "consul"),
							),
						}},
					},
					{
						Name:         "replication_raft",
						ScenarioName: "replication",
						Attributes: cty.ObjectVal(map[string]cty.Value{
							"something":     cty.StringVal("thing"),
							"another_thing": cty.StringVal("another"),
						}),
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "centos"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "bundle"),
								NewElement("distro", "amz"),
								NewElement("edition", "ent.hsm"),
								NewElement("primary_backend", "raft"),
							),
						}},
					},
					{
						Name: "smoke",
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "rhel"),
								NewElement("edition", "ent.hsm"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "centos"),
								NewElement("edition", "ent.hsm"),
							),
							NewVector(
								NewElement("arch", "amd64"),
								NewElement("artifact_source", "artifactory"),
								NewElement("artifact_type", "package"),
								NewElement("distro", "amz"),
								NewElement("edition", "ent.hsm"),
							),
						}},
					},
					{
						Name:           "alias",
						ScenarioFilter: "upgrade arch:amd64 artifact_source:artifactory artifact_type:package distro:rhel edition:ent",
					},
				},
			},
			filter: &pb.Sample_Filter{
				Subsets: []*pb.Sample_Subset_ID{
					{
						Name: "replication_consul",
					},
					{
						Name: "alias",
					},
					{
						Name: "smoke",
					},
				},
				ExcludeSubsets: []*pb.Sample_Subset_ID{
					{
						Name: "alias",
					},
					{
						Name: "replication_consul",
					},
				},
			},
			expected: []*SampleSubset{
				{
					Name: "smoke",
					Matrix: &Matrix{Vectors: []*Vector{
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "rhel"),
							NewElement("edition", "ent.hsm"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "centos"),
							NewElement("edition", "ent.hsm"),
						),
						NewVector(
							NewElement("arch", "amd64"),
							NewElement("artifact_source", "artifactory"),
							NewElement("artifact_type", "package"),
							NewElement("distro", "amz"),
							NewElement("edition", "ent.hsm"),
						),
					}},
				},
			},
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.EqualValues(t, test.expected, test.in.filterSubsets(test.filter))
		})
	}
}
