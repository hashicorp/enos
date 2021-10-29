package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runner returns a default instance of the Enos CLI integration test runner.
func runner(t *testing.T) *testRunner {
	path, ok := os.LookupEnv("ENOS_BINARY_PATH")
	if !ok {
		t.Error("ENOS_BINARY_PATH has not been set")
		t.Fail()
	}

	return &testRunner{
		BinPath: path,
		Env:     os.Environ(),
	}
}

// ensureAcc ensures that acceptance test mode is enables, otherwise it skips.
func ensureAcc(t *testing.T) {
	t.Helper()

	if acc, ok := os.LookupEnv("ENOS_ACC"); ok {
		if acc == "1" || acc == "true" {
			return
		}
	}

	t.Skip("Skipping because ENOS_ACC has not been set")
}

// ensureAccCLI ensures that acceptance test mode is enabled and that a path to
// a test binary has been set.
func ensureAccCLI(t *testing.T) {
	t.Helper()

	ensureAcc(t)

	if path, ok := os.LookupEnv("ENOS_BINARY_PATH"); ok {
		if path != "" {
			return
		}
	}

	t.Skip("Skipping because ENOS_BINARY_PATH has not been set")
}

// testRunner is the Enos CLI integration test runner
type testRunner struct {
	BinPath string
	Env     []string
}

// RunCmd runs an Enos sub-command
func (t *testRunner) RunCmd(ctx context.Context, subCommand string) ([]byte, error) {
	path, err := filepath.Abs(t.BinPath)
	if err != nil {
		return nil, nil
	}
	return exec.CommandContext(ctx, path, strings.Split(subCommand, " ")...).CombinedOutput()
}
