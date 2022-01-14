package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func ensureAcc(t *testing.T) {
	t.Helper()
	if !hasEnosACC() {
		t.Skip("Skipping because ENOS_ACC has not been set")
	}
}

func hasEnosACC() bool {
	if acc, ok := os.LookupEnv("ENOS_ACC"); ok {
		if acc == "1" || acc == "true" {
			return true
		}
	}

	return false
}

func ensureEnosCLI(t *testing.T) {
	t.Helper()
	if !hasEnosCLI() {
		t.Skip("Skipping because ENOS_BINARY_PATH has not been set")
	}
}

func hasEnosCLI() bool {
	if path, ok := os.LookupEnv("ENOS_BINARY_PATH"); ok {
		if path != "" {
			return true
		}
	}

	return false
}

func ensureTerraformCLI(t *testing.T) {
	t.Helper()
	if !hasTerraformCLI() {
		t.Skip("Skipping because terraform binary could not be found in the PATH")
	}
}

func hasTerraformCLI() bool {
	p, err := exec.LookPath("terraform")
	if err != nil || p == "" {
		return false
	}

	return true
}

type acceptanceRunnerOpt func(*acceptanceRunner)

func newAcceptanceRunner(t *testing.T, opts ...acceptanceRunnerOpt) *acceptanceRunner {
	t.Helper()

	r := &acceptanceRunner{
		env: os.Environ(),
	}
	r.enosBinPath, _ = os.LookupEnv("ENOS_BINARY_PATH")
	r.tfBinPath, _ = exec.LookPath("terraform")

	for _, opt := range opts {
		opt(r)
	}

	r.validate(t)

	return r
}

func skipUnlessTerraformCLI() acceptanceRunnerOpt {
	return func(r *acceptanceRunner) {
		r.skipUnlessTerraformCLI = true
	}
}

// acceptanceRunner is the Enos CLI integration test runner
type acceptanceRunner struct {
	enosBinPath            string
	tfBinPath              string
	env                    []string
	skipUnlessTerraformCLI bool
}

// run runs an Enos sub-command
func (r *acceptanceRunner) run(ctx context.Context, subCommand string) ([]byte, error) {
	path, err := filepath.Abs(r.enosBinPath)
	if err != nil {
		return nil, nil
	}
	return exec.CommandContext(ctx, path, strings.Split(subCommand, " ")...).CombinedOutput()
}

func (r *acceptanceRunner) validate(t *testing.T) {
	t.Helper()
	ensureAcc(t)
	ensureEnosCLI(t)
	if r.skipUnlessTerraformCLI {
		ensureTerraformCLI(t)
	}
}
