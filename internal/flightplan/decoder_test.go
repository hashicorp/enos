package flightplan

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	hcl "github.com/hashicorp/hcl/v2"
)

func Test_Decoder_parseDir(t *testing.T) {
	t.Parallel()

	newDecoder := func(dir string) *Decoder {
		path, err := filepath.Abs(filepath.Join("./tests", dir))
		require.NoError(t, err)

		return NewDecoder(
			WithDecoderDirectory(path),
		)
	}

	t.Run("malformed enos.hcl", func(t *testing.T) {
		decoder := newDecoder("parse_dir_fail_malformed_config")
		diags := decoder.Parse()
		require.True(t, diags.HasErrors())
		require.Equal(t, hcl.DiagError, diags[0].Severity)
	})

	t.Run("no matching configuration files", func(t *testing.T) {
		decoder := newDecoder("parse_dir_pass_no_matching_names")
		diags := decoder.Parse()
		require.False(t, diags.HasErrors())
		require.Equal(t, 0, len(decoder.parser.Files()))
	})

	t.Run("two matching files", func(t *testing.T) {
		decoder := newDecoder("parse_dir_pass_two_matching_names")
		diags := decoder.Parse()
		require.False(t, diags.HasErrors())
		require.Equal(t, 2, len(decoder.parser.Files()))
	})
}

func Test_Decoder_Decode(t *testing.T) {
	for _, test := range []struct {
		desc     string
		hcl      string
		expected *FlightPlan
	}{
		{
			desc: "static scenario",
			hcl:  `scenario "basic" { }`,
			expected: &FlightPlan{
				Scenarios: []*Scenario{
					{
						Name: "basic",
					},
				},
			},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			decoder := NewDecoder()
			diags := decoder.parseHCL([]byte(test.hcl), "decoder-test.hcl")
			require.False(t, diags.HasErrors())

			fp, moreDiags := decoder.Decode()
			require.False(t, moreDiags.HasErrors())
			require.EqualValues(t, test.expected.Scenarios, fp.Scenarios)
		})
	}
}
