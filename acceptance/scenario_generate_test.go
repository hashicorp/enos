package acceptance

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAcc_Cmd_Scenario_Generate tests that a scenario can generate into the
// appropriate terraform module and CLI configuration.
func TestAcc_Cmd_Scenario_Generate(t *testing.T) {
	enos := newAcceptanceRunner(t)

	tmpDir, err := os.MkdirTemp("", "enos.generate.out")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	for _, test := range []struct {
		dir  string
		args string
		uid  string
		noRc bool
	}{
		{
			"scenario_generate_pass_0",
			"test foo:matrixfoo",
			fmt.Sprintf("%x", sha256.Sum256([]byte("test [foo:matrixfoo]"))),
			false,
		},
		{
			"scenario_generate_pass_0",
			"test foo:matrixbar",
			fmt.Sprintf("%x", sha256.Sum256([]byte("test [foo:matrixbar]"))),
			false,
		},
		{
			"scenario_generate_pass_backend",
			"",
			fmt.Sprintf("%x", sha256.Sum256([]byte("test"))),
			false,
		},
		{
			"scenario_generate_pass_cloud",
			"",
			fmt.Sprintf("%x", sha256.Sum256([]byte("test"))),
			false,
		},
		{
			"scenario_generate_step_vars",
			"step_vars distro:rhel arch:arm",
			fmt.Sprintf("%x", sha256.Sum256([]byte("step_vars [arch:arm distro:rhel]"))),
			true,
		},
	} {
		t.Run(fmt.Sprintf("%s %s", test.dir, test.args), func(t *testing.T) {
			// NOTE: Right now we're just testing that the generate command
			// outputs the files in the right place with the correct names.
			// Validation and execution are handled by other tests.
			outDir := filepath.Join(tmpDir, test.dir)
			err := os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario generate --chdir %s --out %s %s", path, outDir, test.args)
			out, err := enos.run(context.Background(), cmd)
			require.NoErrorf(t, err, string(out))
			s, err := os.Open(filepath.Join(outDir, test.uid, "scenario.tf"))
			require.NoError(t, err)
			s.Close()
			rc, err := os.Open(filepath.Join(outDir, test.uid, "terraform.rc"))
			if test.noRc {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			rc.Close()
		})
	}
}
