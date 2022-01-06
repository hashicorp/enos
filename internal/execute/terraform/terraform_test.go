package terraform

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test_TerraformSubCommandArgExpansion tests that the terraform sub-command
// only expands arguments that are supported and does it correctly.
func Test_TerraformSubCommandArgExpansion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	type testCase struct {
		cmd          Command
		expectedArgs []string
	}

	verifyCmd := func(t *testing.T, test *testCase, cfg *Config) {
		t.Helper()

		// Expand the terraform command to a *Command
		tfCommand, err := test.cmd.Command(cfg)
		require.NoError(t, err)

		// Expand it to an *exec.Command
		cmd := tfCommand.Cmd(ctx)

		// Make sure we have the right command and args
		require.EqualValues(t, test.expectedArgs, cmd.Args)

		// Make sure we've configured the right path the Terraform CLI config
		if cfg.ConfigPath == "" {
			for _, v := range cmd.Env {
				require.False(t, strings.Contains(v, "TF_CLI_CONFIG_FILE"))
			}
		} else {
			found := false
			for _, v := range cmd.Env {
				if v == "TF_CLI_CONFIG_FILE=/Users/enos/.terraformrc" {
					found = true
					break
				}
			}
			require.True(t, found)
		}

		// Make sure our execution directory is correct
		require.Equal(t, cfg.DirPath, cmd.Dir)

		// Make sure the env is correct
		env := []string{}
		for k, v := range cfg.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(env)
		env = append(os.Environ(), env...)
		require.EqualValues(t, env, cmd.Env)
	}

	t.Run("default config", func(t *testing.T) {
		cfg := NewConfig()
		for _, test := range []*testCase{
			{
				Init,
				[]string{"terraform", "init"},
			},
			{
				Validate,
				[]string{"terraform", "validate"},
			},
			{
				Plan,
				[]string{"terraform", "plan"},
			},
			{
				Apply,
				[]string{"terraform", "apply"},
			},
			{
				Destroy,
				[]string{"terraform", "destroy"},
			},
			{
				Exec,
				[]string{"terraform", ""}, // Our exec sub-command should be missing by default
			},
		} {
			t.Run(test.cmd.String(), func(t *testing.T) {
				verifyCmd(t, test, cfg)
			})
		}
	})

	t.Run("all config options", func(t *testing.T) {
		cfg := NewConfig(
			WithBinPath("/bin/terraform"),
			WithConfigPath("/Users/enos/.terraformrc"),
			WithDirPath("/Users/enos/scenarios/test"),
			WithEnv(map[string]string{"FOO": "BAR", "BAZ": "QUX"}),
			WithExecSubCommand("state show"),
			WithAutoApprove(),
			WithBackupStateFilePath("/Users/enos/scenarios/test/state.bak"),
			WithCompactWarnings(),
			WithLockTimeout(5*time.Second),
			WithNoBackend(),
			WithNoInput(),
			WithNoColor(),
			WithNoLock(),
			WithNoDownload(),
			WithNoRefresh(),
			WithOutPath("/Users/enos/scenarios/test/outpath"),
			WithParallelism(5),
			WithRefreshOnly(),
			WithStateFilePath("/Users/enos/scenarios/test/state"),
			WithUpgrade(),
		)

		for _, test := range []*testCase{
			{
				Init,
				[]string{"/bin/terraform", "init", "-lock-timeout=5.000000s", "-backend=false", "-no-color", "-get=false", "-input=false", "-lock=false", "-upgrade"},
			},
			{
				Validate,
				[]string{"/bin/terraform", "validate", "-no-color"},
			},
			{
				Plan,
				[]string{"/bin/terraform", "plan", "-compact-warnings", "-refresh=false", "-no-color", "-lock=false", "-input=false", "-lock-timeout=5.000000s", "-out=/Users/enos/scenarios/test/outpath", "-parallelism=5", "-refresh-only", "-state=/Users/enos/scenarios/test/state"},
			},
			{
				Apply,
				[]string{"/bin/terraform", "apply", "-auto-approve", "-backup=/Users/enos/scenarios/test/state.bak", "-compact-warnings", "-lock=false", "-lock-timeout=5.000000s", "-input=false", "-parallelism=5", "-state=/Users/enos/scenarios/test/state"},
			},
			{
				Destroy,
				[]string{"/bin/terraform", "destroy", "-auto-approve", "-backup=/Users/enos/scenarios/test/state.bak", "-compact-warnings", "-lock=false", "-lock-timeout=5.000000s", "-input=false", "-parallelism=5", "-state=/Users/enos/scenarios/test/state"},
			},
			{
				Exec,
				[]string{"/bin/terraform", "state", "show"},
			},
		} {
			t.Run(test.cmd.String(), func(t *testing.T) {
				verifyCmd(t, test, cfg)
			})
		}
	})
}
