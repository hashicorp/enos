// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

func Test_SampleFuncAll(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in         *SampleFrame
		expected   *SampleObservation
		shouldFail bool
	}{
		"nil frame": {
			in:         nil,
			shouldFail: true,
		},
		"nil filter": {
			in:         &SampleFrame{},
			shouldFail: true,
		},
		"no subset frames": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 0,
					MaxElements: 10,
				},
			},
			expected: &SampleObservation{
				SampleFrame: &SampleFrame{
					Filter: &pb.Sample_Filter{
						MinElements: 0,
						MaxElements: 10,
					},
				},
				SubsetObservations: SampleSubsetObservations{},
			},
			shouldFail: true,
		},
		"frames with no matrix": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 0,
					MaxElements: 10,
				},
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
			expected: &SampleObservation{
				SampleFrame: &SampleFrame{
					Filter: &pb.Sample_Filter{
						MinElements: 0,
						MaxElements: 10,
					},
				},
				SubsetObservations: SampleSubsetObservations{
					"foo": {
						SampleSubsetFrame: &SampleSubsetFrame{
							SampleSubset: &SampleSubset{
								Name: "foo",
							},
						},
					},
					"foo_alias": {
						SampleSubsetFrame: &SampleSubsetFrame{
							SampleSubset: &SampleSubset{
								Name:         "alias",
								ScenarioName: "foo",
							},
						},
					},
				},
			},
		},
		"incompatible filter": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 0,
					MaxElements: 1, // sample all expects 2
				},
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
			shouldFail: true,
		},
		"frames with matrix": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: 10,
					Percentage:  -1,
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						SampleSubset: &SampleSubset{
							Name: "foo",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						SampleSubset: &SampleSubset{
							Name: "bar",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			expected: &SampleObservation{
				SampleFrame: &SampleFrame{
					Filter: &pb.Sample_Filter{
						MinElements: 1,
						MaxElements: 10,
						Percentage:  -1,
					},
				},
				SubsetObservations: SampleSubsetObservations{
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
		},
		"mixed frames": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 0,
					MaxElements: 10,
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						SampleSubset: &SampleSubset{
							Name: "foo",
						},
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
							Name:         "baz_alias",
							ScenarioName: "baz",
						},
					},
					"bar": {
						SampleSubset: &SampleSubset{
							Name: "bar",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			expected: &SampleObservation{
				SampleFrame: &SampleFrame{
					Filter: &pb.Sample_Filter{
						MinElements: 0,
						MaxElements: 10,
					},
				},
				SubsetObservations: SampleSubsetObservations{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"baz":       {},
					"baz_alias": {},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			//nolint:gosec// G404 we're using a weak random number generator because secure random
			// numbers are not needed for this use case.
			r := rand.New(rand.NewSource(78910))
			obs, err := SampleFuncAll(context.Background(), test.in, r)
			if test.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.EqualValues(t, test.in, obs.SampleFrame)
				require.Equal(t, test.expected.SubsetObservations.Size(), obs.SubsetObservations.Size())
				for name, subObs := range test.expected.SubsetObservations {
					if subObs.Matrix != nil {
						require.True(
							t,
							subObs.Matrix.Equal(obs.SubsetObservations[name].Matrix),
							subObs.Matrix.SymmetricDifferenceUnordered(obs.SubsetObservations[name].Matrix),
						)
					}
				}
			}
		})
	}
}

func Test_SampleFuncStratified(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in         *SampleFrame
		expected   *SampleObservation
		shouldFail bool
	}{
		"nil frame": {
			in:         nil,
			shouldFail: true,
		},
		"nil filter": {
			in:         &SampleFrame{},
			shouldFail: true,
		},
		"no subset frames": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 0,
					MaxElements: 10,
				},
			},
			expected: &SampleObservation{
				SampleFrame: &SampleFrame{
					Filter: &pb.Sample_Filter{
						MinElements: 0,
						MaxElements: 10,
					},
				},
				SubsetObservations: SampleSubsetObservations{},
			},
			shouldFail: true,
		},
		"frames with no matrix max": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: 2,
				},
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
					"bar": {
						SampleSubset: &SampleSubset{
							Name: "foo",
						},
					},
				},
			},
			expected: &SampleObservation{
				SampleFrame: &SampleFrame{
					Filter: &pb.Sample_Filter{
						MinElements: 1,
						MaxElements: 2,
					},
				},
				SubsetObservations: SampleSubsetObservations{
					"foo":       {},
					"foo_alias": {},
				},
			},
		},
		"frames with no matrix pct round down": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: 3,
					Percentage:  49, // %49 of 3 == 1.47 => 1
				},
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
					"bar": {
						SampleSubset: &SampleSubset{
							Name: "foo",
						},
					},
				},
			},
			expected: &SampleObservation{
				SampleFrame: &SampleFrame{
					Filter: &pb.Sample_Filter{
						MinElements: 1,
						MaxElements: 3,
						Percentage:  49,
					},
				},
				SubsetObservations: SampleSubsetObservations{
					"foo": {},
				},
			},
		},
		"frames with no matrix pct round up": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: 3,
					Percentage:  50, // %50 of 3 == 1.50 => 2
				},
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
					"bar": {
						SampleSubset: &SampleSubset{
							Name: "foo",
						},
					},
				},
			},
			expected: &SampleObservation{
				SampleFrame: &SampleFrame{
					Filter: &pb.Sample_Filter{
						MinElements: 1,
						MaxElements: 3,
						Percentage:  50,
					},
				},
				SubsetObservations: SampleSubsetObservations{
					"foo":       {},
					"foo_alias": {},
				},
			},
		},
		"frames with matrix round down": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: -1, // no upper bound
					Percentage:  37, // 37% of 4 = 1.48 = 1
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						SampleSubset: &SampleSubset{
							Name: "foo",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						SampleSubset: &SampleSubset{
							Name: "bar",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			expected: &SampleObservation{
				SubsetObservations: SampleSubsetObservations{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
				},
			},
		},
		"frames with matrix round up": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 0,
					MaxElements: -1, // no upper bound
					Percentage:  38, // 38% of 4 = 1.52 = 2
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						SampleSubset: &SampleSubset{
							Name: "foo",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						SampleSubset: &SampleSubset{
							Name: "bar",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			expected: &SampleObservation{
				SampleFrame: &SampleFrame{
					Filter: &pb.Sample_Filter{
						MinElements: 0,
						MaxElements: -1,
						Percentage:  38,
					},
				},
				SubsetObservations: SampleSubsetObservations{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "consul")),
						}},
					},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
		},
		"width first": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 0,
					MaxElements: 4,
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						SampleSubset: &SampleSubset{
							Name: "foo",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "consul")),
						}},
					},
					"baz": {
						SampleSubset: &SampleSubset{
							Name: "baz",
						},
					},
					"baz_alias": {
						SampleSubset: &SampleSubset{
							Name:         "baz_alias",
							ScenarioName: "baz",
						},
					},
					"bar": {
						SampleSubset: &SampleSubset{
							Name: "bar",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
			expected: &SampleObservation{
				SubsetObservations: SampleSubsetObservations{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "consul")),
						}},
					},
					"baz":       {},
					"baz_alias": {},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
		},
		"deep with remainders": {
			in: &SampleFrame{
				Filter: &pb.Sample_Filter{
					MinElements: 1,
					MaxElements: -1,
					Percentage:  78, // 78% of 12 = 9.36 => 9
				},
				SubsetFrames: SampleSubsetFrames{
					"foo": {
						SampleSubset: &SampleSubset{
							Name: "foo",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "aarch64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "aarch64"), NewElement("primary_backend", "raft")),
						}},
					},
					"baz": {
						SampleSubset: &SampleSubset{
							Name: "baz",
						},
					},
					"baz_alias": {
						SampleSubset: &SampleSubset{
							Name:         "baz_alias",
							ScenarioName: "baz",
						},
					},
					"bar": {
						SampleSubset: &SampleSubset{
							Name: "bar",
						},
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "consul")),
						}},
					},
				},
			},
			expected: &SampleObservation{
				SampleFrame: &SampleFrame{
					Filter: &pb.Sample_Filter{
						MinElements: 0,
						MaxElements: -1,
						Percentage:  78, // 78% of 12 = 9.36 => 9
					},
				},
				SubsetObservations: SampleSubsetObservations{
					"foo": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "aarch64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
						}},
					},
					"baz":       {},
					"baz_alias": {},
					"bar": {
						Matrix: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "consul")),
							NewVector(NewElement("arch", "amd64"), NewElement("primary_backend", "raft")),
							NewVector(NewElement("arch", "arm64"), NewElement("primary_backend", "raft")),
						}},
					},
				},
			},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			//nolint:gosec// G404 we're using a weak random number generator because secure random
			// numbers are not needed for this use case.
			r := rand.New(rand.NewSource(78910))
			obs, err := SampleFuncPurposiveStratified(context.Background(), test.in, r)
			if test.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.EqualValues(t, test.in, obs.SampleFrame)
				require.Equal(t, test.expected.SubsetObservations.Size(), obs.SubsetObservations.Size())
				for name, expObs := range test.expected.SubsetObservations {
					if expObs.Matrix != nil {
						gotObs, ok := obs.SubsetObservations[name]
						require.Truef(t, ok, "expected subset observation %s, got %v", name, obs.SubsetObservations)
						require.Truef(t, expObs.Matrix.Equal(gotObs.Matrix), fmt.Sprintf(
							"expected matrix vectors for %s: \n%s\ngot matrix vectors: \n%s\ndifference: \n%s\n",
							name,
							expObs.Matrix.String(),
							gotObs.Matrix.String(),
							expObs.Matrix.SymmetricDifferenceUnordered(gotObs.Matrix).String(),
						))
					}
				}
			}
		})
	}
}
