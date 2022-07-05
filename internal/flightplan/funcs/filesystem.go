package funcs

import (
	"os"
	"path/filepath"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// AbsPathFunc constructs a function that converts a filesystem path to an
// absolute path. It takes basePath that is equal to the decoders working
// directory, that way relative paths are relative to the working dir.
func AbsPathFunc(basePath string) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "path",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			path := args[0].AsString()
			if filepath.IsAbs(path) {
				return cty.StringVal(filepath.ToSlash(path)), nil
			}

			if basePath != "" {
				path = filepath.Join(basePath, path)
			}
			absPath, err := filepath.Abs(path)
			return cty.StringVal(filepath.ToSlash(absPath)), err
		},
	})
}

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

// FileFunc reads the contents of the file at the path given. The file must
// be valid UTF-8.
var FileFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "path",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		f, err := os.ReadFile(args[0].AsString())
		return cty.StringVal(string(f)), err
	},
})
