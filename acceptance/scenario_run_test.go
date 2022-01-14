package acceptance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAcc_Cmd_Scenario_Run tests that a scenario can be generated and validated
// with Terraform.
func TestAcc_Cmd_Scenario_Run(t *testing.T) {
	enos := newAcceptanceRunner(t, skipUnlessTerraformCLI())

	tmpDir, err := os.MkdirTemp("/tmp", "enos.launch")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	for _, test := range []struct {
		dir    string
		outDir string
	}{
		{
			dir:    "scenario_generate_pass_0",
			outDir: tmpDir,
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario run --chdir %s --out %s", path, test.outDir)
			out, err := enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))
			_, err = os.Open(filepath.Join(test.outDir, "test", "terraform.tfstate"))
			require.NoError(t, err)
		})
	}
}
