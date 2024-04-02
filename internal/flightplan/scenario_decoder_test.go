// Copyright (c) HashiCorp, Inc.
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

func Test_DecodedScenarioBlocks_Diagnostics(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       DecodedScenarioBlocks
		expected hcl.Diagnostics
	}{
		"nil": {
			in:       nil,
			expected: nil,
		},
		"none": {
			in:       DecodedScenarioBlocks{},
			expected: nil,
		},
		"one": {
			in: DecodedScenarioBlocks{
				{
					Diagnostics: hcl.Diagnostics{
						{
							Summary: "one",
						},
					},
				},
			},
			expected: hcl.Diagnostics{
				{
					Summary: "one",
				},
			},
		},
		"multiple": {
			in: DecodedScenarioBlocks{
				{
					Diagnostics: hcl.Diagnostics{
						{
							Summary: "one",
						},
						{
							Summary: "two",
						},
					},
				},
			},
			expected: hcl.Diagnostics{
				{
					Summary: "one",
				},
				{
					Summary: "two",
				},
			},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.EqualValues(t, test.expected, test.in.Diagnostics())
		})
	}
}

func Test_DecodedScenarioBlocks_Scenarios(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       DecodedScenarioBlocks
		expected []*Scenario
	}{
		"nil": {
			in:       nil,
			expected: nil,
		},
		"none": {
			in:       DecodedScenarioBlocks{},
			expected: nil,
		},
		"one": {
			in: DecodedScenarioBlocks{
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
			in: DecodedScenarioBlocks{
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
			require.EqualValues(t, test.expected, test.in.Scenarios())
		})
	}
}

func Test_DecodedScenarioBlocks_CombinedMatrix(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       DecodedScenarioBlocks
		expected *Matrix
	}{
		"nil": {
			in:       nil,
			expected: nil,
		},
		"none": {
			in:       DecodedScenarioBlocks{},
			expected: nil,
		},
		"one": {
			in: DecodedScenarioBlocks{
				{
					DecodedMatrices: &DecodedMatrices{
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
			in: DecodedScenarioBlocks{
				{
					DecodedMatrices: &DecodedMatrices{
						FinalProduct: &Matrix{Vectors: []*Vector{
							NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
							NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
							NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
							NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
						}},
					},
				},
				{
					DecodedMatrices: &DecodedMatrices{
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

func Test_ScenarioDecoder_filterScenarioBlocks(t *testing.T) {
	t.Parallel()

	rng := hcl.Range{
		Filename: "foo.hcl",
		Start:    hcl.Pos{Line: 1, Column: 1, Byte: 1},
		End:      hcl.Pos{Line: 2, Column: 1, Byte: 2},
	}

	for desc, test := range map[string]struct {
		decoder    *ScenarioDecoder
		blocks     []*hcl.Block
		expected   DecodedScenarioBlocks
		shouldFail bool
	}{
		"no blocks": {
			decoder:  &ScenarioDecoder{},
			blocks:   nil,
			expected: nil,
		},
		"no filter": {
			decoder: &ScenarioDecoder{
				EvalContext:    &hcl.EvalContext{},
				DecodeTarget:   DecodeTargetScenariosComplete,
				ScenarioFilter: nil,
			},
			blocks: []*hcl.Block{
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
			expected: DecodedScenarioBlocks{
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
			},
			blocks: []*hcl.Block{
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
			expected: DecodedScenarioBlocks{
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
			},
			blocks: []*hcl.Block{
				{
					Type:        "scenario",
					Labels:      []string{""},
					LabelRanges: []hcl.Range{rng, rng},
				},
			},
			expected: DecodedScenarioBlocks{
				{
					Name:         "",
					DecodeTarget: DecodeTargetScenariosComplete,
					Diagnostics:  hcl.Diagnostics{{Severity: hcl.DiagError}},
					Block: &hcl.Block{
						Type:        "scenario",
						Labels:      []string{""},
						LabelRanges: []hcl.Range{rng, rng},
					},
				},
			},
		},
		"too many labels": {
			decoder: &ScenarioDecoder{
				EvalContext:  &hcl.EvalContext{},
				DecodeTarget: DecodeTargetScenariosComplete,
			},
			blocks: []*hcl.Block{
				{
					Type:        "scenario",
					Labels:      []string{"foo", "bar"},
					LabelRanges: []hcl.Range{rng, rng},
				},
			},
			expected: DecodedScenarioBlocks{
				{
					Name:         "foo",
					DecodeTarget: DecodeTargetScenariosComplete,
					Diagnostics:  hcl.Diagnostics{{Severity: hcl.DiagError}},
					Block: &hcl.Block{
						Type:        "scenario",
						Labels:      []string{"foo", "bar"},
						LabelRanges: []hcl.Range{rng, rng},
					},
				},
			},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			res := test.decoder.filterScenarioBlocks(test.blocks)
			if test.shouldFail {
				require.GreaterOrEqual(t, 1, len(res.Diagnostics()), "expected result to have diagnostics")
			} else {
				require.Len(t, test.expected, len(res))
				for i := range test.expected {
					require.Equal(t, test.expected[i].Name, res[i].Name)
					require.Equal(t, test.expected[i].Block, res[i].Block)
				}
			}
		})
	}
}
