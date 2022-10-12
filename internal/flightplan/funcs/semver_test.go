package funcs

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestServerConstraint(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		version    string
		constraint string
		expected   bool
	}{
		"handles leading v": {
			"v1.13.0",
			">=1.11.0",
			true,
		},
		"does what we expect with with prerelases": {
			"1.13.0-dev1",
			">=1.11.0",
			false,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			val, err := SemverConstraint.Call([]cty.Value{
				cty.StringVal(test.version), cty.StringVal(test.constraint),
			})

			require.NoError(t, err)
			require.Equal(t, cty.True, val.Equals(cty.BoolVal(test.expected)))
		})
	}
}
