package acceptance

import (
	"context"
	"crypto/sha256"
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
		dir  string
		args string
		uids []string
	}{
		{
			"scenario_e2e_aws",
			"e2e",
			[]string{
				fmt.Sprintf("%x", sha256.Sum256([]byte("e2e [aws_region:east distro:rhel]"))),
				fmt.Sprintf("%x", sha256.Sum256([]byte("e2e [aws_region:west distro:ubuntu]"))),
			},
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			outDir := filepath.Join(tmpDir, test.dir)
			err := os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
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
				for _, uid := range test.uids {
					bytes, err := os.ReadFile(filepath.Join(outDir, uid, "scenario.tf"))
					if err != nil {
						t.Logf("unable to read failed scenario's generated module: %s", err.Error())
					} else {
						t.Logf("%s/scenario.tf\n (%s) %s", uid, test.args, string(bytes))
					}

					bytes, err = os.ReadFile(filepath.Join(outDir, uid, "terraform.rc"))
					if err != nil {
						t.Logf("unable to read failed scenario's generated cli config: %s", err.Error())
					} else {
						t.Logf("%s/terraform.rc (%s)\n %s", uid, test.args, string(bytes))
					}
				}

				// Lets try one more time to destroy resources that might have been
				// created
				out, err := enos.run(context.Background(), fmt.Sprintf("scenario destroy --chdir %s --out %s", path, outDir))
				assert.NoErrorf(t, err, string(out))
			})

			cmd := fmt.Sprintf("scenario run --chdir %s --out %s", path, outDir)
			out, err := enos.run(context.Background(), cmd)
			if err != nil {
				failed = true
			}
			require.NoError(t, err, string(out))
		})
	}
}
