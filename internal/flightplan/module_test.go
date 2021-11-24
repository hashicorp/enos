package flightplan

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test_Module_EvalContext_Functions tests a few built-in functions to ensure
// that they're available when the module blocks are evaluated.
func Test_Module_EvalContext_Functions(t *testing.T) {
	t.Parallel()

	modulePath, err := filepath.Abs("./tests/simple_module")
	require.NoError(t, err)

	// This isn't an exhaustive test of all functions, but we should have
	// access to functions in the base resource context.
	for _, test := range []struct {
		desc     string
		expr     string
		expected string
	}{
		{
			desc:     "upper",
			expr:     `upper("low")`,
			expected: "LOW",
		},
		{
			desc:     "trimsuffix",
			expr:     `trimsuffix("something.com", ".com")`,
			expected: "something",
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			hcl := fmt.Sprintf(`
module "backend" {
  source = "%s"
  something = %s
}

scenario "basic" {
  step "first" {
    module = module.backend
  }
}`, modulePath, test.expr)

			cwd, err := os.Getwd()
			require.NoError(t, err)
			decoder, err := NewDecoder(WithDecoderBaseDir(cwd))
			require.NoError(t, err)
			diags := decoder.parseHCL([]byte(hcl), "decoder-test.hcl")
			require.False(t, diags.HasErrors(), diags.Error())

			fp, moreDiags := decoder.Decode()
			require.False(t, moreDiags.HasErrors(), moreDiags.Error())

			require.Equal(t, 1, len(fp.Modules))
			v, ok := fp.Modules[0].Attrs["something"]
			require.True(t, ok)

			require.Equal(t, test.expected, v.AsString())
		})
	}
}
