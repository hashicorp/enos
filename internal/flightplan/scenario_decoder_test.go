// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"testing"

	"github.com/stretchr/testify/require"

	hcl "github.com/hashicorp/hcl/v2"
)

func Test_NewScenarioDecoder(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		opts       []ScenarioDecoderOpt
		shouldFail bool
	}{
		"target unset": {
			opts:       []ScenarioDecoderOpt{WithScenarioDecoderDecodeTarget(DecodeTargetUnset)},
			shouldFail: true,
		},
		"target out of range": {
			opts:       []ScenarioDecoderOpt{WithScenarioDecoderDecodeTarget(DecodeTargetAll + 1)},
			shouldFail: true,
		},
		"target names": {
			opts: []ScenarioDecoderOpt{WithScenarioDecoderDecodeTarget(DecodeTargetScenariosNamesNoVariants)},
		},
		"target matrix": {
			opts: []ScenarioDecoderOpt{WithScenarioDecoderDecodeTarget(DecodeTargetScenariosMatrixOnly)},
		},
		"target expand": {
			opts: []ScenarioDecoderOpt{WithScenarioDecoderDecodeTarget(DecodeTargetScenariosNamesExpandVariants)},
		},
		"target complete": {
			opts: []ScenarioDecoderOpt{WithScenarioDecoderDecodeTarget(DecodeTargetScenariosComplete)},
		},
		"target all": {
			opts: []ScenarioDecoderOpt{WithScenarioDecoderDecodeTarget(DecodeTargetAll)},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			_, err := NewScenarioDecoder(test.opts...)
			if test.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_ScenarioBlocks_Scenarios(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       ScenarioBlocks
		expected []*Scenario
	}{
		"nil": {
			in:       nil,
			expected: nil,
		},
		"none": {
			in:       ScenarioBlocks{},
			expected: nil,
		},
		"one": {
			in: ScenarioBlocks{
				{
					Scenarios: []*Scenario{
						{
							Name: "one-one",
						},
						{
							Name: "one-two",
						},
					},
				},
			},
			expected: []*Scenario{
				{
					Name: "one-one",
				},
				{
					Name: "one-two",
				},
			},
		},
		"multiple": {
			in: ScenarioBlocks{
				{
					Scenarios: []*Scenario{
						{
							Name: "one-one",
						},
						{
							Name: "one-two",
						},
					},
				},
				{
					Scenarios: []*Scenario{
						{
							Name: "two-one",
						},
						{
							Name: "two-two",
						},
					},
				},
			},
			expected: []*Scenario{
				{
					Name: "one-one",
				},
				{
					Name: "one-two",
				},
				{
					Name: "two-one",
				},
				{
					Name: "two-two",
				},
			},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, test.in.Scenarios())
		})
	}
}

func Test_ScenarioBlocks_CombinedMatrix(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       ScenarioBlocks
		expected *Matrix
	}{
		"nil": {
			in:       nil,
			expected: nil,
		},
		"none": {
			in:       ScenarioBlocks{},
			expected: nil,
		},
		"one": {
			in: ScenarioBlocks{
				{
					MatrixBlock: &MatrixBlock{
						FinalProduct: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
							NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
							NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
							NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
						}},
					},
				},
			},
			expected: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
		},
		"multiple": {
			in: ScenarioBlocks{
				{
					MatrixBlock: &MatrixBlock{
						FinalProduct: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
							NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
							NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
							NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
						}},
					},
				},
				{
					MatrixBlock: &MatrixBlock{
						FinalProduct: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
							NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
							NewVector(NewElement("backend", "postgres"), NewElement("arch", "aarch65")),
							NewVector(NewElement("backend", "mysql"), NewElement("arch", "s309x")),
						}},
					},
				},
			},
			expected: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "postgres"), NewElement("arch", "aarch65")),
				NewVector(NewElement("backend", "mysql"), NewElement("arch", "s309x")),
			}},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			if test.expected == nil {
				require.Nil(t, test.in.CombinedMatrix())
			} else {
				require.True(t, test.expected.EqualUnordered(test.in.CombinedMatrix()))
			}
		})
	}
}

func Test_ScenarioDecoderIterator_filterHCLBlocks(t *testing.T) {
	t.Parallel()

	rng := hcl.Range{
		Filename: "foo.hcl",
		Start:    hcl.Pos{Line: 1, Column: 1, Byte: 1},
		End:      hcl.Pos{Line: 2, Column: 1, Byte: 2},
	}

	for desc, test := range map[string]struct {
		decoder    *ScenarioDecoder
		expected   ScenarioBlocks
		shouldFail bool
	}{
		"no blocks": {
			decoder:  &ScenarioDecoder{},
			expected: nil,
		},
		"no filter": {
			decoder: &ScenarioDecoder{
				EvalContext:    &hcl.EvalContext{},
				DecodeTarget:   DecodeTargetScenariosComplete,
				ScenarioFilter: nil,
				Blocks: []*hcl.Block{
					{
						Type:        "scenario",
						Labels:      []string{"foo"},
						LabelRanges: []hcl.Range{rng, rng},
					},
					{
						Type:        "scenario",
						Labels:      []string{"bar"},
						LabelRanges: []hcl.Range{rng, rng},
					},
				},
			},
			expected: ScenarioBlocks{
				{
					Name:         "foo",
					DecodeTarget: DecodeTargetScenariosComplete,
					Block: &hcl.Block{
						Type:        "scenario",
						Labels:      []string{"foo"},
						LabelRanges: []hcl.Range{rng, rng},
					},
				},
				{
					Name:         "bar",
					DecodeTarget: DecodeTargetScenariosComplete,
					Block: &hcl.Block{
						Type:        "scenario",
						Labels:      []string{"bar"},
						LabelRanges: []hcl.Range{rng, rng},
					},
				},
			},
		},
		"unmatched label": {
			decoder: &ScenarioDecoder{
				EvalContext:  &hcl.EvalContext{},
				DecodeTarget: DecodeTargetScenariosComplete,
				ScenarioFilter: &ScenarioFilter{
					Name: "foo",
				},
				Blocks: []*hcl.Block{
					{
						Type:        "scenario",
						Labels:      []string{"foo"},
						LabelRanges: []hcl.Range{rng, rng},
					},
					{
						Type:        "scenario",
						Labels:      []string{"bar"},
						LabelRanges: []hcl.Range{rng, rng},
					},
				},
			},
			expected: ScenarioBlocks{
				{
					Name:         "foo",
					DecodeTarget: DecodeTargetScenariosComplete,
					Block: &hcl.Block{
						Type:        "scenario",
						Labels:      []string{"foo"},
						LabelRanges: []hcl.Range{rng, rng},
					},
				},
			},
		},
		"no labels": {
			decoder: &ScenarioDecoder{
				EvalContext:  &hcl.EvalContext{},
				DecodeTarget: DecodeTargetScenariosComplete,
				Blocks: []*hcl.Block{
					{
						Type:        "scenario",
						Labels:      []string{""},
						LabelRanges: []hcl.Range{rng, rng},
					},
				},
			},
			expected: nil,
		},
		"too many labels": {
			decoder: &ScenarioDecoder{
				EvalContext:  &hcl.EvalContext{},
				DecodeTarget: DecodeTargetScenariosComplete,
				Blocks: []*hcl.Block{
					{
						Type:        "scenario",
						Labels:      []string{"foo", "bar"},
						LabelRanges: []hcl.Range{rng, rng},
					},
				},
			},
			expected: nil,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			iter := test.decoder.Iterator()
			diags := iter.filterHCLBlocks()
			if test.shouldFail {
				require.GreaterOrEqual(t, 1, len(diags), "expected result to have diagnostics")
			} else {
				require.Len(t, iter.scenarioBlocks, len(test.expected))
				for i := range test.expected {
					require.Equal(t, test.expected[i].Name, iter.scenarioBlocks[i].Name)
					require.Equal(t, test.expected[i].Block, iter.scenarioBlocks[i].Block)
				}
			}
		})
	}
}

func Test_ScenarioDecoderIterator_filterScenarioBlocksWithMatrixBlocks(t *testing.T) {
	t.Parallel()

	// NOTE: This test assumes that the input set has already been filtered by
	// name during scenario block decode and that the matrix vectors have already
	// been decoded by filterHCLBlocks() and decodeMatrix(). As such we just verify
	// that when given a filter without a name but with variants we exclude blocks
	// whose matrices don't intersect with our filter.
	for desc, test := range map[string]struct {
		in       ScenarioBlocks
		expected ScenarioBlocks
		filter   *ScenarioFilter
	}{
		"filter without name but variants": {
			ScenarioBlocks{
				{
					Name: "foo",
					MatrixBlock: &MatrixBlock{FinalProduct: &Matrix{Vectors: []*Vector{
						NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
						NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
					}}},
				},
				{
					Name: "bar",
					MatrixBlock: &MatrixBlock{FinalProduct: &Matrix{Vectors: []*Vector{
						NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
						NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
					}}},
				},
				{
					Name:        "baz",
					MatrixBlock: nil,
				},
			},
			ScenarioBlocks{
				{
					Name: "foo",
					MatrixBlock: &MatrixBlock{FinalProduct: &Matrix{Vectors: []*Vector{
						NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
						NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
					}}},
				},
				{
					Name: "bar",
					MatrixBlock: &MatrixBlock{FinalProduct: &Matrix{Vectors: []*Vector{
						NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
						NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
					}}},
				},
			},
			&ScenarioFilter{
				Name:    "test",
				Include: NewVector(NewElement("arch", "amd64")),
			},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			iter := &ScenarioDecoderIterator{
				filter:         test.filter,
				scenarioBlocks: test.in,
			}

			iter.filterScenarioBlocksWithMatrixBlocks()
			require.Len(t, iter.scenarioBlocks, len(test.expected))
			for i := range test.expected {
				require.Equal(t, test.expected[i].Name, iter.scenarioBlocks[i].Name)
				if test.expected[i].Matrix() == nil {
					require.Nilf(t, iter.scenarioBlocks[i].Matrix(), "expected nil, got: %s", iter.scenarioBlocks[i].Matrix().String())
				} else {
					require.True(t, test.expected[i].Matrix().EqualUnordered(iter.scenarioBlocks[i].Matrix()))
				}
			}
		})
	}
}
