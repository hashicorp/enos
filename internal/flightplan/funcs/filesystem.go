package funcs

import (
	"path/filepath"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// AbsPathFunc constructs a function that converts a filesystem path to an absolute path
var AbsPathFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "path",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		absPath, err := filepath.Abs(args[0].AsString())
		return cty.StringVal(filepath.ToSlash(absPath)), err
	},
})

// JoinPathFunc constructs a function that converts joins two paths
var JoinPathFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "root",
			Type: cty.String,
		},
		{
			Name: "path",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(filepath.ToSlash(filepath.Join(
			args[0].AsString(), args[1].AsString()),
		)), nil
	},
})
