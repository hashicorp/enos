// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcc_Cmd_Version(t *testing.T) {
	t.Parallel()

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
		t.Run(test.cmd, func(t *testing.T) {
			t.Parallel()

			enos := newAcceptanceRunner(t)
			out, _, err := enos.run(context.Background(), test.cmd)
			require.NoError(t, err)
			require.True(t, test.out.Match(out))
		})
	}
}
