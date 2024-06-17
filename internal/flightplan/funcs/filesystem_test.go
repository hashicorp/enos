// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package funcs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestAbsPathFunc(t *testing.T) {
	tmpDir := t.TempDir()
	t.Parallel()

	for name, test := range map[string]struct {
		basePath func() string
		path     string
		absPath  func() string
	}{
		"absolute_with_irrelevant_base": {
			basePath: func() string { return "/my/current/working/dir" },
			path:     "/some/absolute/path/file.txt",
			absPath:  func() string { return "/some/absolute/path/file.txt" },
		},
		"absolute_no_base": {
			basePath: func() string { return "" },
			path:     "/some/absolute/path/file.txt",
			absPath:  func() string { return "/some/absolute/path/file.txt" },
		},
		"relative_no_base": {
			basePath: func() string {
				wd, err := os.Getwd()
				require.NoError(t, err)

				return wd
			},
			path: "./some/relative",
			absPath: func() string {
				wd, err := os.Getwd()
				require.NoError(t, err)
				abs, err := filepath.Abs(filepath.Join(wd, "./some/relative"))
				require.NoError(t, err)

				return abs
			},
		},
		"relative_base": {
			basePath: func() string { return tmpDir },
			path:     "./some/relative",
			absPath: func() string {
				abs, err := filepath.Abs(filepath.Join(tmpDir, "./some/relative"))
				require.NoError(t, err)

				return abs
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			basePath := test.basePath()
			absPath, err := AbsPathFunc(basePath).Call([]cty.Value{cty.StringVal(test.path)})
			require.NoError(t, err)
			require.Equal(t, test.absPath(), absPath.AsString())
		})
	}
}

func TestFileFunc(t *testing.T) {
	tmpDir := t.TempDir()
	t.Parallel()

	for name, test := range map[string]struct {
		basePath func() string
		path     func() string
		contents string
	}{
		"absolute_with_irrelevant_base": {
			basePath: func() string { return "/my/current/working/dir" },
			path: func() string {
				p, err := filepath.Abs("./testdata/test_file_func.txt")
				require.NoError(t, err)

				return p
			},
			contents: "static\n",
		},
		"absolute_no_base": {
			basePath: func() string { return "" },
			path: func() string {
				p, err := filepath.Abs("./testdata/test_file_func.txt")
				require.NoError(t, err)

				return p
			},
			contents: "static\n",
		},
		"relative": {
			basePath: func() string {
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test_file_func.txt"), []byte("dynamic"), 0o755))
				return tmpDir
			},
			path:     func() string { return "./test_file_func.txt" },
			contents: "dynamic",
		},
		"relative_no_base": {
			basePath: func() string { return "" },
			path:     func() string { return "./testdata/test_file_func.txt" },
			contents: "static\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			basePath := test.basePath()
			contents, err := FileFunc(basePath).Call([]cty.Value{cty.StringVal(test.path())})
			require.NoError(t, err)
			require.Equal(t, test.contents, contents.AsString())
		})
	}
}
