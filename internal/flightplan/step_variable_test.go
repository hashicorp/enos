package flightplan

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/hcl/v2/hcldec"
)

func testMakeStepVarTraversal(parts ...string) cty.Value {
	traversal := hcl.Traversal{}
	for i, part := range parts {
		if i == 0 {
			traversal = append(traversal, hcl.TraverseRoot{Name: part})
			continue
		}
		traversal = append(traversal, hcl.TraverseAttr{Name: part})
	}

	return StepVariableVal(&StepVariable{Traversal: traversal})
}

func testMakeStepVarValue(val cty.Value) cty.Value {
	return StepVariableVal(&StepVariable{Value: val})
}

func Test_StepVariableType_Decode(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		desc            string
		body            string
		ctx             *hcl.EvalContext
		value           cty.Value
		expectValue     bool
		expectTraversal bool
		fail            bool
	}{
		{
			desc:        "known primitive value",
			body:        `val = "foo"`,
			ctx:         &hcl.EvalContext{},
			expectValue: true,
			value: cty.ObjectVal(map[string]cty.Value{
				"val": testMakeStepVarValue(cty.StringVal("foo")),
			}),
		},
		{
			// this could happen with value chaining through multiple steps
			desc: "known stepvar value",
			body: `val = "foo"`,
			ctx: &hcl.EvalContext{Variables: map[string]cty.Value{
				"foo": testMakeStepVarValue(cty.StringVal("foo")),
			}},
			expectValue: true,
			value: cty.ObjectVal(map[string]cty.Value{
				// should inherit the actual value of the capsule
				"val": testMakeStepVarValue(cty.StringVal("foo")),
			}),
		},
		{
			desc: "valid absolute traversal ref",
			body: `val = step.foo.ref.thing`,
			ctx: &hcl.EvalContext{Variables: map[string]cty.Value{
				"step": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{}),
				}),
			}},
			expectTraversal: true,
			value:           testMakeStepVarTraversal("step", "foo", "ref", "thing"),
		},
		{
			desc: "no step in context",
			body: `val = step.foo.ref.thing`,
			ctx:  &hcl.EvalContext{},
			fail: true,
		},
		{
			desc: "no matching step in context",
			body: `val = step.foo.ref.thing`,
			ctx: &hcl.EvalContext{Variables: map[string]cty.Value{
				"step": cty.ObjectVal(map[string]cty.Value{
					"bar": cty.ObjectVal(map[string]cty.Value{}),
				}),
			}},
			fail: true,
		},
		{
			desc: "variables",
			body: `val = var.aws_availability_zones`,
			ctx: &hcl.EvalContext{Variables: map[string]cty.Value{
				"var": cty.ObjectVal(map[string]cty.Value{
					"aws_availability_zones": cty.ListVal([]cty.Value{cty.StringVal("*")}),
				}),
			}},
			value: testMakeStepVarValue(cty.ListVal([]cty.Value{cty.StringVal("*")})),
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(test.body), "", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}
			spec := hcldec.ObjectSpec{
				"val": &hcldec.AttrSpec{
					Name:     "val",
					Type:     StepVariableType,
					Required: true,
				},
			}

			val, diags := hcldec.Decode(file.Body, spec, test.ctx)
			if diags.HasErrors() != test.fail {
				t.Fatalf("expected: %t, got: %t, err: %s", test.fail, diags.HasErrors(), diags.Error())
			}

			if test.expectValue {
				require.EqualValues(t, test.value, val)
			}

			if test.expectTraversal {
				attr, ok := val.AsValueMap()["val"]
				require.True(t, ok, "'val' was not found in the object")
				testMostlyEqualStepVar(t, test.value, attr)
			}
		})
	}
}
