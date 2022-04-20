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

// TestAcc_Cmd_Scenario_Validate tests that a scenario can be generated and validated
// with Terraform.
func TestAcc_Cmd_Scenario_Validate(t *testing.T) {
	enos := newAcceptanceRunner(t, skipUnlessTerraformCLI())

	tmpDir, err := os.MkdirTemp("/tmp", "enos.validate")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	for _, test := range []struct {
		dir  string
		args string
		uid  string
	}{
		{
			"scenario_generate_pass_0",
			"test foo:matrixfoo",
			fmt.Sprintf("%x", sha256.Sum256([]byte("test [foo:matrixfoo]"))),
		},
		{
			"scenario_generate_pass_0",
			"test foo:matrixbar",
			fmt.Sprintf("%x", sha256.Sum256([]byte("test [foo:matrixbar]"))),
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
		})
	}
}
