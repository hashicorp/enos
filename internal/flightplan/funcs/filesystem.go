// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package funcs

import (
	"os"
	"path/filepath"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// AbsPathFunc constructs a function that converts a filesystem path to an absolute path. It takes
// basePath that is equal to the decoders working directory, that way relative paths are relative
// to the working dir.
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
			str, err := absolutePathRelativeToBase(basePath, path)

			return cty.StringVal(str), err
		},
	})
}

// JoinPathFunc constructs a function that converts joins two paths.
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

// FileFunc constructs a function that FileFunc reads the contents of the file at the path given.
// The file must be valid UTF-8. It takes basePath that is equal to the decoders working directory,
// that way relative paths are relative to the working dir.
func FileFunc(basePath string) function.Function {
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
			abs, err := absolutePathRelativeToBase(basePath, path)
			if err != nil {
				return cty.StringVal(""), err
			}

			f, err := os.ReadFile(abs)

			return cty.StringVal(string(f)), err
		},
	})
}

// Take a basePath and another path and return the absolute path. If the path is absolute we'll
// return it unchanged. If it's relative and a basePath is not blank we'll make the path absolute
// realitve to the basePath, otherwise it will be absolute relative to the current directory.
func absolutePathRelativeToBase(base, path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.ToSlash(path), nil
	}

	if base != "" {
		path = filepath.Join(base, path)
	}
	absPath, err := filepath.Abs(path)

	return filepath.ToSlash(absPath), err
}
