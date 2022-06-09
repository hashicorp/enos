package flightplan

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

var variableSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "description"},
		{Name: "default"},
		{Name: "type"},
		{Name: "sensitive"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: blockTypeValidation},
	},
}

// Variable represents a "variable" block in a module or file.
type Variable struct {
	Name           string
	Description    string
	Sensitive      bool
	Default        cty.Value
	SetValue       cty.Value
	Type           cty.Type
	ConstraintType cty.Type
}

// VariableValue is a user supplied variable value
type VariableValue struct {
	Expr      hcl.Expression
	Range     hcl.Range
	Source    VariableValueSource
	EnvVarRaw string
}

type VariableValueSource int

const (
	VariableValueSourceUnknown VariableValueSource = iota
	VariableValueSourceVarsFile
	VariableValueSourceEnvVar
)

const EnvVarPrefix = "ENOS_VAR_"

// NewVariable returns a new Variable
func NewVariable() *Variable {
	return &Variable{}
}

// decode takes in an HCL block of a variable and any set variable values and
// and decodes itself.
func (v *Variable) decode(block *hcl.Block, values map[string]*VariableValue) hcl.Diagnostics {
	var diags hcl.Diagnostics

	content, moreDiags := block.Body.Content(variableSchema)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	v.Name = block.Labels[0]

	if attr, ok := content.Attributes["description"]; ok {
		moreDiags := gohcl.DecodeExpression(attr.Expr, nil, &v.Description)
		diags = diags.Extend(moreDiags)
	}

	if attr, ok := content.Attributes["type"]; ok {
		ty, moreDiags := typeexpr.TypeConstraint(attr.Expr)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			ty = cty.DynamicPseudoType
		}
		v.ConstraintType = ty
		v.Type = ty.WithoutOptionalAttributesDeep()
	}

	if attr, ok := content.Attributes["sensitive"]; ok {
		// NOTE: We don't actually do anything with sensitive variables yet
		// but we're reserving them here.
		moreDiags := gohcl.DecodeExpression(attr.Expr, nil, &v.Sensitive)
		diags = diags.Extend(moreDiags)
	}

	if attr, ok := content.Attributes["default"]; ok {
		val, moreDiags := attr.Expr.Value(nil)
		diags = diags.Extend(moreDiags)

		if v.ConstraintType != cty.NilType {
			var err error
			val, err = convert.Convert(val, v.ConstraintType)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid default value for variable",
					Detail:   fmt.Sprintf("This default value is not compatible with the variable's type constraint: %s.", err),
					Subject:  attr.Expr.Range().Ptr(),
				})
				val = cty.DynamicVal
			}
		}

		v.Default = val
	}

	if setVal, ok := values[v.Name]; ok {
		switch setVal.Source {
		case VariableValueSourceEnvVar:
			// Env vars are tricky. They might be a string or a complex type.
			// If we know they're a string we'll decode them as a string literal.
			// If we don't know the type information we'll try the expression
			// and if it fails then we'll assume it's a string literal.
			if v.Type == cty.String || v.ConstraintType == cty.String {
				v.SetValue = cty.StringVal(setVal.EnvVarRaw)
			} else {
				expr, exprDiags := hclsyntax.ParseExpression(
					[]byte(setVal.EnvVarRaw),
					setVal.Range.Filename,
					hcl.InitialPos,
				)

				val, moreExprDiags := expr.Value(nil)
				exprDiags = exprDiags.Extend(moreExprDiags)
				if !exprDiags.HasErrors() {
					v.SetValue = val
				} else {
					v.SetValue = cty.StringVal(setVal.EnvVarRaw)
				}
			}
		default:
			// We have a value that isn't from the environment so we'll get
			// the value of the expression.
			val, moreDiags := setVal.Expr.Value(nil)
			diags = diags.Extend(moreDiags)
			if moreDiags.HasErrors() {
				v.SetValue = cty.DynamicVal
				return diags
			}
			v.SetValue = val
		}

		if v.ConstraintType != cty.NilType {
			val, err := convert.Convert(v.SetValue, v.ConstraintType)
			v.SetValue = val
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid value for variable",
					Detail:   fmt.Sprintf("This default value is not compatible with the variable's type constraint: %s.", err),
					Subject:  setVal.Range.Ptr(),
				})
				v.SetValue = cty.DynamicVal
			}
		}
	}

	return diags
}

// Value returns either the user-supplied value or the default. If no values have
// been set it will always return a NilVal.
func (v *Variable) Value() cty.Value {
	if v.SetValue != cty.NilVal {
		return v.SetValue
	}

	if v.Default != cty.NilVal {
		return v.Default
	}

	return cty.NilVal
}
