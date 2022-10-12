package funcs

import (
	semver "github.com/Masterminds/semver/v3"
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
			Name: "constraint",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		version, constraint := args[0].AsString(), args[1].AsString()

		v, err := semver.NewVersion(version)
		if err != nil {
			return cty.NullVal(cty.Bool), err
		}

		c, err := semver.NewConstraint(constraint)
		if err != nil {
			return cty.NullVal(cty.Bool), err
		}

		return cty.BoolVal(c.Check(v)), nil
	},
})
