package acceptance

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

	for _, testDir := range []string{
		"scenario_generate_pass_0",
		"scenario_generate_pass_backend",
		"scenario_generate_pass_cloud",
	} {
		t.Run(testDir, func(t *testing.T) {
			// NOTE: Right now we're just testing that the generate command
			// outputs the files in the right place with the correct names.
			// Validation and execution are handled by other tests.
			outDir := filepath.Join(tmpDir, testDir)
			err := os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
			path, err := filepath.Abs(filepath.Join("./scenarios", testDir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario generate --chdir %s --out %s", path, outDir)
			out, err := enos.run(context.Background(), cmd)
			require.NoErrorf(t, err, string(out))
			s, err := os.Open(filepath.Join(outDir, "test/scenario.tf"))
			require.NoError(t, err)
			s.Close()
			rc, err := os.Open(filepath.Join(outDir, "test/terraform.rc"))
			require.NoError(t, err)
			rc.Close()
		})
	}
}
