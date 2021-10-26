package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Runner returns a default instance of the Enos CLI integration test runner.
func Runner(t *testing.T) *TestRunner {
	path, ok := os.LookupEnv("ENOS_BINARY_PATH")
	if !ok {
		t.Error("ENOS_BINARY_PATH has not been set")
		t.Fail()
	}

	return &TestRunner{
		BinPath: path,
		Env:     os.Environ(),
	}
}

// EnsureAcc ensures that acceptance test mode is enables, otherwise it skips.
func EnsureAcc(t *testing.T) {
	t.Helper()

	if acc, ok := os.LookupEnv("ENOS_ACC"); ok {
		if acc == "1" || acc == "true" {
			return
		}
	}

	t.Skip("Skipping because ENOS_ACC has not been set")
}

// EnsureAccCLI ensures that acceptance test mode is enabled and that a path to
// a test binary has been set.
func EnsureAccCLI(t *testing.T) {
	t.Helper()

	EnsureAcc(t)

	if path, ok := os.LookupEnv("ENOS_BINARY_PATH"); ok {
		if path != "" {
			return
		}
	}

	t.Skip("Skipping because ENOS_BINARY_PATH has not been set")
}

// TestRunner is the Enos CLI integration test runner
type TestRunner struct {
	BinPath string
	Env     []string
}

// RunCmd runs an Enos sub-command
func (t *TestRunner) RunCmd(ctx context.Context, subCommand string) ([]byte, error) {
	path, err := filepath.Abs(t.BinPath)
	if err != nil {
		return nil, nil
	}
	return exec.CommandContext(ctx, path, strings.Split(subCommand, " ")...).Output()
}
