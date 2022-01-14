package cmd

import (
	"context"
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
		dir    string
		outDir string
	}{
		{
			dir:    "scenario_generate_pass_0",
			outDir: tmpDir,
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			// NOTE: Right now we're just testing that the generate command
			// outputs the files in the right place with the correct names.
			// Validation and execution are handled by other tests.
			path, err := filepath.Abs(filepath.Join("./integration_tests", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario generate --chdir %s --out %s", path, test.outDir)
			_, err = enos.run(context.Background(), cmd)
			require.NoError(t, err)
			s, err := os.Open(filepath.Join(test.outDir, "test/scenario.tf"))
			require.NoError(t, err)
			s.Close()
			rc, err := os.Open(filepath.Join(test.outDir, "test/terraform.rc"))
			require.NoError(t, err)
			rc.Close()
		})
	}
}
