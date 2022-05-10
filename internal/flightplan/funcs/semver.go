package funcs

import (
	"github.com/Masterminds/semver"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// SemverConstraint takes a semantic version and a constraint and returns a
// boolean of whether or not the constaint has been met.
var SemverConstraint = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "version",
			Type: cty.String,
		},
		{
			Name: "constaint",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		v, err := semver.NewVersion(args[0].AsString())
		if err != nil {
			return cty.NullVal(cty.Bool), err
		}

		c, err := semver.NewConstraint(args[1].AsString())
		if err != nil {
			return cty.NullVal(cty.Bool), err
		}

		return cty.BoolVal(c.Check(v)), nil
	},
})
