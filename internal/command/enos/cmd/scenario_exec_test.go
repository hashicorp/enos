package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAcc_Cmd_Scenario_Exec tests that a raw Terrform command can be passed
// to a scenario's Terraform.
func TestAcc_Cmd_Scenario_Exec(t *testing.T) {
	enos := newAcceptanceRunner(t, skipUnlessTerraformCLI())

	tmpDir, err := os.MkdirTemp("/tmp", "enos.exec")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

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
			path, err := filepath.Abs(filepath.Join("./integration_tests", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario validate --chdir %s --out %s", path, test.outDir)
			out, err := enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))
			cmd = fmt.Sprintf("scenario launch --chdir %s --out %s", path, test.outDir)
			out, err = enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))
			cmd = fmt.Sprintf("scenario exec --cmd 'state show' --chdir %s --out %s", path, test.outDir)
			out, err = enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))
			cmd = fmt.Sprintf("scenario destroy --chdir %s --out %s", path, test.outDir)
			out, err = enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))
		})
	}
}
