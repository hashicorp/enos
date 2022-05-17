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

// TestAcc_Cmd_Scenario_Output tests that a Terraform output command succeeds
func TestAcc_Cmd_Scenario_Output(t *testing.T) {
	// TODO: This test is pretty dumb in that it just looks for a truty exit,
	// we'd need proper machine output and parsing to really validate it
	enos := newAcceptanceRunner(t, skipUnlessTerraformCLI())

	tmpDir, err := os.MkdirTemp("/tmp", "enos.exec")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	for _, test := range []struct {
		dir  string
		args string
		uid  string
	}{
		{
			"scenario_generate_step_vars",
			"step_vars distro:rhel arch:arm",
			fmt.Sprintf("%x", sha256.Sum256([]byte("step_vars [arch:arm distro:rhel]"))),
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			outDir := filepath.Join(tmpDir, test.dir)
			err := os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario validate --chdir %s --out %s", path, outDir)
			out, err := enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))
			_, err = os.Open(filepath.Join(outDir, test.uid, ".terraform/modules/modules.json"))
			require.NoError(t, err)
			cmd = fmt.Sprintf("scenario launch --chdir %s --out %s", path, outDir)
			out, err = enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))
			_, err = os.Open(filepath.Join(outDir, test.uid, "terraform.tfstate"))
			require.NoError(t, err)
			cmd = fmt.Sprintf(`scenario output --name step_reference_unknown --chdir %s --out %s %s`, path, outDir, test.args)
			out, err = enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))
			cmd = fmt.Sprintf(`scenario output --chdir %s --out %s %s`, path, outDir, test.args)
			out, err = enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))
			cmd = fmt.Sprintf("scenario destroy --chdir %s --out %s", path, outDir)
			out, err = enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))
		})
	}
}
