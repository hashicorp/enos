// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func Test_SchemalessBlock_Decode(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		desc     string
		body     string
		ctx      *hcl.EvalContext
		expected *SchemalessBlock
	}{
		{
			desc: "big old nested guy",
			body: `
provider "is" "anything" {
  attr = "1"

  no_label {
    attr = "1"
    child_no_label {
      attr = "2"
      deep_child_no_label {
        attr = boop
      }
    }
  }

  one_label "one" {
    attr = "1"
    child_one_label "two" {
      attr = "2"
      deep_child_one_label "three" {
        attr = boop
      }
    }
  }
}`,
			ctx: &hcl.EvalContext{
				Variables: map[string]cty.Value{
					"boop": cty.StringVal("beep"),
				},
			},
			expected: &SchemalessBlock{
				Type:   "provider",
				Labels: []string{"is", "anything"},
				Attrs:  map[string]cty.Value{"attr": cty.StringVal("1")},
				Children: []*SchemalessBlock{
					{
						Type:  "no_label",
						Attrs: map[string]cty.Value{"attr": cty.StringVal("1")},
						Children: []*SchemalessBlock{
							{
								Type:  "child_no_label",
								Attrs: map[string]cty.Value{"attr": cty.StringVal("2")},
								Children: []*SchemalessBlock{
									{
										Type:     "deep_child_no_label",
										Attrs:    map[string]cty.Value{"attr": cty.StringVal("beep")},
										Children: []*SchemalessBlock{},
									},
								},
							},
						},
					},
					{
						Type:   "one_label",
						Labels: []string{"one"},
						Attrs:  map[string]cty.Value{"attr": cty.StringVal("1")},
						Children: []*SchemalessBlock{
							{
								Type:   "child_one_label",
								Labels: []string{"two"},
								Attrs:  map[string]cty.Value{"attr": cty.StringVal("2")},
								Children: []*SchemalessBlock{
									{
										Type:     "deep_child_one_label",
										Labels:   []string{"three"},
										Attrs:    map[string]cty.Value{"attr": cty.StringVal("beep")},
										Children: []*SchemalessBlock{},
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			file, diags := hclsyntax.ParseConfig([]byte(test.body), "in.hcl", hcl.InitialPos)
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}
			files := map[string]*hcl.File{"in.hcl": file}

			body, ok := file.Body.(*hclsyntax.Body)
			require.True(t, ok)

			csb := NewSchemalessBlock()
			diags = csb.Decode(body.Blocks[0].AsHCLBlock(), test.ctx)
			if len(diags) > 0 {
				err := testDiagsToError(files, diags)
				require.NoError(t, err)
			}
			require.EqualValues(t, test.expected, csb)
		})
	}
}

func Test_SchemalessBlock_Roundtrip(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		desc     string
		ctx      *hcl.EvalContext
		expected *SchemalessBlock
	}{
		{
			desc: "big old nested guy",
			ctx: &hcl.EvalContext{
				Variables: map[string]cty.Value{
					"boop": cty.StringVal("beep"),
				},
			},
			expected: &SchemalessBlock{
				Type:   "provider",
				Labels: []string{"is", "anything"},
				Attrs:  map[string]cty.Value{"attr": cty.StringVal("1")},
				Children: []*SchemalessBlock{
					{
						Type:   "no_label",
						Labels: []string{},
						Attrs:  map[string]cty.Value{"attr": cty.StringVal("1")},
						Children: []*SchemalessBlock{
							{
								Type:   "child_no_label",
								Labels: []string{},
								Attrs:  map[string]cty.Value{"attr": cty.StringVal("2")},
								Children: []*SchemalessBlock{
									{
										Type:     "deep_child_no_label",
										Labels:   []string{},
										Attrs:    map[string]cty.Value{"attr": cty.StringVal("boop")},
										Children: []*SchemalessBlock{},
									},
								},
							},
						},
					},
					{
						Type:   "one_label",
						Labels: []string{"one"},
						Attrs:  map[string]cty.Value{"attr": cty.StringVal("1")},
						Children: []*SchemalessBlock{
							{
								Type:   "child_one_label",
								Labels: []string{"two"},
								Attrs:  map[string]cty.Value{"attr": cty.StringVal("2")},
								Children: []*SchemalessBlock{
									{
										Type:     "deep_child_one_label",
										Labels:   []string{"three"},
										Attrs:    map[string]cty.Value{"attr": cty.StringVal("boop")},
										Children: []*SchemalessBlock{},
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			val := test.expected.ToCtyValue()
			got := NewSchemalessBlock()
			require.NoError(t, got.FromCtyValue(val))
			require.EqualValues(t, test.expected, got)
		})
	}
}
