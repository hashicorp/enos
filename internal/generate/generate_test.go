// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test_maybeUpdateRelativeSourcePaths verifies that we rewrite source paths
// relative to the filepath defined in the enos.hcl. We also need to ensure
// that we don't rewrite when given sources that aren't paths.
// NOTE: when run on macOS where $TMPDIR is usually symlinked to /private/var,
// we implicitly test that symlinked paths work.
func Test_MaybeUpdateRelativeSourcePaths(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "scenarios/test")
	moduleDir := filepath.Join(baseDir, "modules/foo")
	outDir := filepath.Join(tmpDir, "generated/out")
	require.NoError(t, os.MkdirAll(moduleDir, 0o755))
	require.NoError(t, os.MkdirAll(outDir, 0o755))
	moduleDirEval, err := filepath.EvalSymlinks(moduleDir)
	require.NoError(t, err)

	for _, test := range []struct {
		desc     string
		source   string
		expected string
		baseDir  string
		outDir   string
	}{
		{
			desc:     "absolute",
			source:   moduleDir,
			expected: moduleDirEval,
		},
		{
			desc:     "relative ./",
			source:   "./modules/foo",
			expected: "../../scenarios/test/modules/foo",
		},
		{
			desc:     "relative ../",
			source:   "../modules/foo",
			baseDir:  baseDir + "/modules",
			expected: "../../scenarios/test/modules/foo",
		},
		{
			desc:     "relative ../../",
			source:   "../../modules/foo",
			baseDir:  moduleDir,
			expected: "../../scenarios/test/modules/foo",
		},
		{
			desc:     "relative same dir",
			source:   "./",
			baseDir:  tmpDir,
			outDir:   tmpDir,
			expected: "./",
		},
		{
			desc:     "relative all common ancestors",
			source:   "./modules/foo",
			baseDir:  baseDir,
			outDir:   baseDir,
			expected: "./modules/foo",
		},
		{
			desc:     "relative commmon ancestor deep",
			source:   "./modules/foo",
			baseDir:  baseDir,
			outDir:   tmpDir,
			expected: "./scenarios/test/modules/foo",
		},
		{
			desc:     "registry",
			source:   "terraform-aws-modules/vpc/aws",
			expected: "terraform-aws-modules/vpc/aws",
		},
		{
			desc:     "github",
			source:   "github.com/hashicorp/example",
			expected: "github.com/hashicorp/example",
		},
		{
			desc:     "bitbucket",
			source:   "bitbucket.org/hashicorp/terraform-consul-aws",
			expected: "bitbucket.org/hashicorp/terraform-consul-aws",
		},
		{
			desc:     "git",
			source:   "git::ssh://username@example.com/storage.git",
			expected: "git::ssh://username@example.com/storage.git",
		},
		{
			desc:     "https",
			source:   "https://example.com/vpc-module?archive=zip",
			expected: "https://example.com/vpc-module?archive=zip",
		},
		{
			desc:     "s3",
			source:   "s3::https://s3-eu-west-1.amazonaws.com/examplecorp-terraform-modules/vpc.zip",
			expected: "s3::https://s3-eu-west-1.amazonaws.com/examplecorp-terraform-modules/vpc.zip",
		},
		{
			desc:     "gcs",
			source:   "gcs::https://www.googleapis.com/storage/v1/modules/foomodule.zip",
			expected: "gcs::https://www.googleapis.com/storage/v1/modules/foomodule.zip",
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			bd := baseDir
			if test.baseDir != "" {
				bd = test.baseDir
			}
			od := outDir
			if test.outDir != "" {
				od = test.outDir
			}

			got, err := maybeUpdateRelativeSourcePaths(test.source, bd, od)
			require.NoError(t, err)
			require.Equal(t, test.expected, got)
		})
	}
}
