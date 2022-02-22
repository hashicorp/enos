package acceptance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAcc_Cmd_Scenario_E2E_AWS does an end-to-end test with AWS
func TestAcc_Cmd_Scenario_E2E_AWS(t *testing.T) {
	enos := newAcceptanceRunner(t,
		skipUnlessTerraformCLI(),
		skipUnlessAWSCredentials(),
		skipUnlessEnosPrivateKey(),
		skipUnlessExtEnabled(),
	)

	tmpDir, err := os.MkdirTemp("/tmp", "enos.aws.e2e")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	for _, test := range []struct {
		dir    string
		outDir string
	}{
		{
			dir:    "scenario_e2e_aws",
			outDir: tmpDir,
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)

			failed := false
			// Make sure we do our best to cleanup any real cloud resources
			t.Cleanup(func() {
				if !failed {
					return
				}

				// Since we failed lets output something useful that might help
				// us debug the problem
				bytes, err := os.ReadFile(filepath.Join(test.outDir, "e2e/scenario.tf"))
				if err != nil {
					t.Logf("unable to read failed scenario's generated module: %s", err.Error())
				} else {
					t.Logf("ec2/scenario.tf\n %s", string(bytes))
				}

				bytes, err = os.ReadFile(filepath.Join(test.outDir, "e2e/terraform.rc"))
				if err != nil {
					t.Logf("unable to read failed scenario's generated cli config: %s", err.Error())
				} else {
					t.Logf("ec2/terraform.rc\n %s", string(bytes))
				}

				// Lets try one more time to destroy resources that might have been
				// created
				out, err := enos.run(context.Background(), fmt.Sprintf("scenario destroy --chdir %s --out %s", path, test.outDir))
				assert.NoErrorf(t, err, string(out))
			})

			cmd := fmt.Sprintf("scenario run --chdir %s --out %s", path, test.outDir)
			out, err := enos.run(context.Background(), cmd)
			if err != nil {
				failed = true
			}
			require.NoError(t, err, string(out))
		})
	}
}
