package generate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test_maybeUpdateRelativeSourcePaths tests convering relative module import
// paths to the correct location.
func Test_maybeUpdateRelativeSourcePaths(t *testing.T) {
	t.Parallel()

	baseDir := "/Users/enos/scenarios"
	outDir := "/tmp/out"
	for _, test := range []struct {
		desc     string
		source   string
		expected string
		baseDir  string
		outDir   string
	}{
		{
			desc:     "absolute",
			source:   "/Users/enos/scenario/modules/foo",
			expected: "/Users/enos/scenario/modules/foo",
		},
		{
			desc:     "relative ./",
			source:   "./modules/foo",
			expected: "../../Users/enos/scenarios/modules/foo",
		},
		{
			desc:     "relative ../",
			source:   "../modules/foo",
			expected: "../../Users/enos/modules/foo",
		},
		{
			desc:     "relative ../../",
			source:   "../modules/foo",
			expected: "../../Users/enos/modules/foo",
		},
		{
			desc:     "relative ./../../",
			source:   "../modules/foo",
			expected: "../../Users/enos/modules/foo",
		},
		{
			desc:     "relative same dir",
			source:   "./",
			baseDir:  "/Users/enos",
			outDir:   "/Users/enos",
			expected: "./",
		},
		{
			desc:     "relative all common ancestors",
			source:   "./modules/foo",
			baseDir:  "/Users/enos",
			outDir:   "/Users/enos",
			expected: "./modules/foo",
		},
		{
			desc:     "relative one common ancestor",
			source:   "./modules/foo",
			baseDir:  "/Users/enos",
			outDir:   "/Users/sone",
			expected: "../enos/modules/foo",
		},
		{
			desc:     "relative two common ancestors",
			source:   "./modules/foo",
			baseDir:  "/Users/enos",
			outDir:   "/Users/enos/out",
			expected: "../modules/foo",
		},
		{
			desc:     "relative commmon ancestor deep",
			source:   "./modules/foo",
			baseDir:  "/Users/enos/projects/enos/scenarios",
			outDir:   "/Users/enos/out",
			expected: "../projects/enos/scenarios/modules/foo",
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
