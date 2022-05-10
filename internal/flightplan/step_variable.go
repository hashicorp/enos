package flightplan

import (
	"fmt"
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// StepVariableType is a cty capsule type that represents "step" variables.
// Step variables might be known values or unknown references to
// to module outputs. Due to the complex nature of these values we have our
// cty Type to carry this information for us.
var StepVariableType cty.Type

// StepVariable is the type encapsulated in StepVariableType
type StepVariable struct {
	Value     cty.Value
	Traversal hcl.Traversal
}

// StepVariableVal returns a new cty.Value of type StepVariableType
func StepVariableVal(stepVar *StepVariable) cty.Value {
	return cty.CapsuleVal(StepVariableType, stepVar)
}

// StepVariableFromVal returns the *StepVariable from a given value.
func StepVariableFromVal(v cty.Value) (*StepVariable, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var stepVar *StepVariable

	if !v.Type().Equals(StepVariableType) {
		return stepVar, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "invalid value",
		})
	}

	return v.EncapsulatedValue().(*StepVariable), diags
}

// absTraversalForExpr is similar to hcl.AbsTraversalForExpr() in that it returns
// an expression as an absolute value. Where it differs is that our implementation
// will use the passed in EvalContext to resolve values in the expression that
// might otherwise be unknown.
// NOTE: This implemenation currently only support expanding the values of keys
// in index expressions. Enos is intended to support passing configuration between
// modules by reference. If you need to perform complex operations on step
// variables you'll need to perform that in the module that is taking the value
// as an input.
func absTraversalForExpr(expr hcl.Expression, ctx *hcl.EvalContext) (hcl.Traversal, hcl.Diagnostics) {
	traversal, diags := hcl.AbsTraversalForExpr(expr)
	if !diags.HasErrors() {
		// We have a valid absolute traversal
		return traversal, diags
	}

	traversal = hcl.Traversal{}

	// If we're here we're dealing with an expression that has neither a known
	// value or a static absolute traversal. We'll attempt to unwrap our expresion
	// and decode unknown values into static values where possible.
	for {
		switch t := expr.(type) {
		case *hclsyntax.ScopeTraversalExpr:
			// We're run into what is likely the root of our traversal. Append
			// what we've got and break our loop as there are no more collection
			// expressions to unwrap.
			return append(t.AsTraversal(), traversal...), nil
		case *hclsyntax.IndexExpr:
			v, err := t.Key.Value(ctx)
			if err != nil {
				return traversal, hcl.Diagnostics{&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "unable to resolve index value",
					Detail:   err.Error(),
					Subject:  t.StartRange().Ptr(),
					Context:  t.SrcRange.Ptr(),
				}}
			}
			// Add our known index value to the traversal and set the next
			// collection expression for unwrapping
			traversal = append(hcl.Traversal{hcl.TraverseIndex{
				SrcRange: t.SrcRange,
				Key:      v,
			}}, traversal...)
			expr = t.Collection
		default:
			return traversal, hcl.Diagnostics{&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("expanding expression for %s is not supported", reflect.TypeOf(t).Name()),
				Subject:  t.StartRange().Ptr(),
				Context:  t.Range().Ptr(),
			}}
		}
	}
}

func init() {
	StepVariableType = cty.CapsuleWithOps("stepvar", reflect.TypeOf(StepVariable{}), &cty.CapsuleOps{
		ExtensionData: func(key any) any {
			switch key {
			case customdecode.CustomExpressionDecoder:
				return customdecode.CustomExpressionDecoderFunc(
					func(expr hcl.Expression, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
						// Step variables are a tricky concept in enos because
						// we're dealing with either a known input value
						// or a reference to a Terraform module output which
						// is unknown. Because unknown reference values are possible
						// we can't rely on normal expressions evaluation and
						// instead do a combination of expression evaluation for
						// known values and static analysis for unknown module
						// output references. We'll do our best to support complex
						// references but they have to be absolute traversals to
						// to "step"'s in the evaluation context.
						var diags hcl.Diagnostics
						stepVar := &StepVariable{
							Value: cty.NilVal,
						}

						// Try and get an absolute value. This should work if
						// the value is knowable: an known value of primitive
						// types or a previously decoded step variable, in the
						// latter case we really only care about copying the
						// contents of the value to avoid nesting stepvars.
						absVal, moreDiags := expr.Value(ctx)
						if !moreDiags.HasErrors() {
							// It's an known value. If it's a stepvar get the
							// value out of it to avoid nesting known stepvars.
							if absVal.Type().Equals(StepVariableType) {
								nested, moreDiags := StepVariableFromVal(absVal)
								diags = diags.Extend(moreDiags)
								if moreDiags.HasErrors() {
									return StepVariableVal(stepVar), diags
								}
								absVal = nested.Value
							}

							stepVar.Value = absVal
							return StepVariableVal(stepVar), diags
						}

						// We have an unknown value. Let's find out if it's a
						// valid traversal to another "step".
						traversal, moreDiags := absTraversalForExpr(expr, ctx)
						if moreDiags.HasErrors() {
							// If it's not an absolute traversal we can't do
							// static analysis.
							return StepVariableVal(stepVar), diags.Extend(moreDiags)
						}

						// It's an absolute traversal. Find out if it's a valid "step" reference.
						if traversal.RootName() != "step" {
							// It's an unknowable value that isn't a reference to
							// a step output.
							return StepVariableVal(stepVar), diags.Append(&hcl.Diagnostic{
								Severity: hcl.DiagError,
								Subject:  traversal.SourceRange().Ptr(),
								Context:  expr.Range().Ptr(),
								Summary:  "step variable is unknowable",
								Detail:   "step variables can only be unknown if the value is a reference to a step module output",
							})
						}

						// Make sure we're referencing a known step.
						steps, err := findEvalContextVariable("step", ctx)
						if err != nil {
							return StepVariableVal(stepVar), diags.Append(&hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  "no previous steps have been defined",
								Subject:  traversal.SourceRange().Ptr(),
								Context:  expr.Range().Ptr(),
							})
						}

						stepName, ok := traversal[1].(hcl.TraverseAttr)
						if !ok {
							return StepVariableVal(stepVar), diags.Append(&hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  "invalid step traversal",
								Subject:  traversal.SourceRange().Ptr(),
								Context:  expr.Range().Ptr(),
							})
						}

						_, ok = steps.AsValueMap()[stepName.Name]
						if !ok {
							return StepVariableVal(stepVar), diags.Append(&hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  fmt.Sprintf("no step named %s has been previously defined", stepName.Name),
								Subject:  stepName.SourceRange().Ptr(),
								Context:  traversal.SourceRange().Ptr(),
							})
						}

						stepVar.Traversal = traversal
						return StepVariableVal(stepVar), diags
					},
				)
			default:
				return nil
			}
		},
		TypeGoString: func(_ reflect.Type) string {
			return "flightplan.StepVariable"
		},
		GoString: func(raw any) string {
			stepVar, _ := raw.(*StepVariable)
			return fmt.Sprintf("flightplan.StepVariable(%#v)", stepVar)
		},
		RawEquals: func(a, b any) bool {
			stepVarA, _ := a.(*StepVariable)
			stepVarB, _ := b.(*StepVariable)
			return (stepVarA.Value == stepVarB.Value) &&
				reflect.DeepEqual(stepVarA.Traversal, stepVarB.Traversal)
		},
	})
}
