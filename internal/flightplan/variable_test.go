package flightplan

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

func Test_Decode_Variable(t *testing.T) {
	fakeRng := hcl.Range{
		Filename: "notreal",
		Start:    hcl.Pos{Line: 1, Column: 1},
		End:      hcl.Pos{Line: 1, Column: 1},
	}

	for _, test := range []struct {
		desc     string
		vars     map[string]*VariableValue
		enosCfg  string
		expected cty.Value
		fail     bool
	}{
		{
			desc: "set",
			vars: map[string]*VariableValue{
				"astring": {
					Value: cty.StringVal("stringval"),
				},
			},
			enosCfg: `
variable "astring" {
  description = "astring desc"
  type = string
  default = "defaultval"
}
`,
			expected: cty.StringVal("stringval"),
		},
		{
			desc: "default",
			enosCfg: `
variable "astring" {
  description = "astring desc"
  type = string
  default = "defaultval"
}
`,
			expected: cty.StringVal("defaultval"),
		},
		{
			desc: "invalid value",
			vars: map[string]*VariableValue{
				"astring": {
					Value: cty.ObjectVal(map[string]cty.Value{
						"invalid": cty.StringVal("val"),
					}),
					Range: fakeRng,
				},
			},

			enosCfg: `
variable "astring" {
  description = "astring desc"
  type = string
  default = "defaultval"
}
`,
			fail: true,
		},
		{
			desc: "complex set",
			vars: map[string]*VariableValue{
				"complex": {
					Value: cty.ObjectVal(map[string]cty.Value{
						"nested": cty.ObjectVal(map[string]cty.Value{
							"numlist": cty.ListVal([]cty.Value{cty.StringVal("foo")}),
						}),
						"abool": cty.BoolVal(true),
					}),
				},
			},
			enosCfg: `
variable "complex" {
  description = "complex desc"
  type = object({
    nested = object({
      numlist = list(string)
	})
	abool = bool
  })
  default = null
}
`,
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"numlist": cty.ListVal([]cty.Value{cty.StringVal("foo")}),
				}),
				"abool": cty.BoolVal(true),
			}),
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			parser := hclparse.NewParser()
			f, diags := parser.ParseHCL([]byte(test.enosCfg), "variable.hcl")
			require.False(t, diags.HasErrors(), diags.Error())
			content, diags := f.Body.Content(&hcl.BodySchema{
				Blocks: []hcl.BlockHeaderSchema{
					{Type: blockTypeVariable, LabelNames: []string{attrLabelNameDefault}},
				},
			})
			require.False(t, diags.HasErrors(), diags.Error())
			block := content.Blocks.OfType(blockTypeVariable)[0]
			variable := NewVariable()
			diags = variable.decode(block, test.vars)
			if test.fail {
				require.Error(t, diags.Errs()[0])
			} else {
				require.False(t, diags.HasErrors(), diags.Error())
				require.Equal(t, test.expected, variable.Value())
			}
		})
	}
}
