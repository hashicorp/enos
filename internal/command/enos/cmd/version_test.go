package cmd

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcc_Cmd_Version(t *testing.T) {
	enos := newAcceptanceRunner(t)

	for _, test := range []struct {
		cmd  string
		out  *regexp.Regexp
		fail bool
	}{
		{
			cmd: "version",
			out: regexp.MustCompile(`\d*\.\d*\.\d*`),
		},
		{
			cmd: "version --all",
			out: regexp.MustCompile(`Enos version: \d*\.\d*\.\d* sha: \w*`),
		},
	} {
		out, err := enos.run(context.Background(), test.cmd)
		require.NoError(t, err)
		require.True(t, test.out.Match(out))
	}
}
