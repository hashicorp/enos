// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

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
		expected   cty.Value
	}{
		"respects prerelases without a constraint range": {
			"v1.13.0-dev1",
			">= 1.11.0-0",
			cty.True,
		},
		"respects prerelases with constraint range": {
			"v1.13.0-dev1",
			">= 1.11.0-0",
			cty.True,
		},
		"respects prerelases with multiple constraints with ranges": {
			"v1.17.0-rc1",
			"< 1.18.0, >= 1.17.0-0",
			cty.True,
		},
		"respects prerelases with multiple constraints without ranges": {
			"v1.17.0-rc1",
			"< 1.18.0, >= 1.17.0",
			cty.False,
		},
		"VAULT-40512: prerelease version constraints not working": {
			"v1.17.0-rc1",
			"<1.19.4-0,>=1.19.0-0 || <1.18.15-0,>=1.18.0-0 || <1.18.0,>=1.17.0-0 || <1.16.26-0",
			cty.True,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			val, err := SemverConstraint.Call([]cty.Value{
				cty.StringVal(test.version), cty.StringVal(test.constraint),
			})

			require.NoError(t, err)
			require.Equal(t, test.expected, val)
		})
	}
}
