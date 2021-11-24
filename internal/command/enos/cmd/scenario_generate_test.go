package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcc_Cmd_Scenario_Generate(t *testing.T) {
	ensureAccCLI(t)
	runner := runner(t)

	tmpDir, err := os.MkdirTemp("", "enos.generate.out")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	for _, test := range []struct {
		dir string
		out string
	}{
		{
			dir: "scenario_generate_pass_0",
			out: filepath.Join(tmpDir, "test.tf"),
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			// NOTE: Right now we're just testing that the generate command
			// outputs the file in the right place with the correct name.
			// Testing the contents would require us to parse the Terraform
			// HCL, which we can't reasonably without requiring Terraform to
			// be installed. When we add support for executing generated things
			// we can build up additional gates for this.
			path, err := filepath.Abs(filepath.Join("./integration_tests", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario generate --chdir %s --out %s", path, tmpDir)
			_, err = runner.RunCmd(context.Background(), cmd)
			require.NoError(t, err)
			_, err = os.Open(test.out)
			require.NoError(t, err)
		})
	}
}
