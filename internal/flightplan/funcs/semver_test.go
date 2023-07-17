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
		"doesn't adhere to prerelases without a constraint range": {
			"v1.13.0-dev1",
			">=1.11.0",
			false,
		},
		"respects prerelases with constraint range": {
			"v1.13.0-dev1",
			">=1.11.0-0",
			true,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			val, err := SemverConstraint.Call([]cty.Value{
				cty.StringVal(test.version), cty.StringVal(test.constraint),
			})

			require.NoError(t, err)
			require.Equal(t, cty.True, val.Equals(cty.BoolVal(test.expected)))
		})
	}
}
