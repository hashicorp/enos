package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcc_Cmd_Scenario_List(t *testing.T) {
	enos := newAcceptanceRunner(t)

	// NOTE: these are a bit brittle as they depend on our output format not
	// changing. When we add machine readable output we should switch to that.
	for _, test := range []struct {
		dir  string
		out  string
		fail bool
		err  error
	}{
		{
			dir: "scenario_list_pass_0",
			out: "",
		},
		{
			dir: "scenario_list_pass_1",
			out: "SCENARIO \n        \ntest    \n",
		},
		{
			dir: "scenario_list_pass_2",
			out: "SCENARIO \n        \nconsul  \nvault   \n",
		},
		{
			dir:  "scenario_list_fail_malformed",
			fail: true,
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			path, err := filepath.Abs(filepath.Join("./integration_tests", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario list --chdir %s", path)
			fmt.Println(path)
			out, err := enos.run(context.Background(), cmd)

			if test.fail {
				if test.err != nil {
					require.ErrorIs(t, err, test.err)
				} else {
					require.Error(t, err)
				}

				if test.out != "" {
					require.Equal(t, test.out, string(out))
				}

				return
			}

			require.NoError(t, err)
			require.Equal(t, test.out, string(out))
		})
	}
}
