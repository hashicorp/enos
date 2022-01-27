package acceptance

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
)

func ensureAcc(t *testing.T) {
	t.Helper()
	if !hasEnosACC() {
		t.Skip("Skipping because ENOS_ACC has not been set. You must set this environment value to a truthy value to execute acceptance tests. Running make test-acc will do this")
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

func ensureExt(t *testing.T) {
	t.Helper()
	if !hasEnosExt() {
		t.Skip("Skipping because ENOS_EXT has not been set. You must set this environment value to a truthy value to execute acceptance tests which require external resources like AWS. Running make test-acc-ext with the appropriate support files and AWS credentials should run these tests.")
	}
}

func hasEnosExt() bool {
	if acc, ok := os.LookupEnv("ENOS_EXT"); ok {
		if acc == "1" || acc == "true" {
			return true
		}
	}

	return false
}

func ensureEnosCLI(t *testing.T) {
	t.Helper()
	if !hasEnosCLI() {
		t.Skip("Skipping because ENOS_BINARY_PATH has not been set. make test-acc will do this for you.")
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
		t.Skip("Skipping because terraform binary could not be found in the PATH. This should be set to the binary version of enos you wish to perform acceptance testing with")
	}
}

func hasTerraformCLI() bool {
	p, err := exec.LookPath("terraform")
	if err != nil || p == "" {
		return false
	}

	return true
}

func ensureAWSCredentials(t *testing.T) {
	t.Helper()
	if !hasAWSCredentials() {
		t.Skip("Skipping because valid AWS credentials could not be resolved. Have you used doormat to get keys?")
	}
}

func hasAWSCredentials() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return false
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return false
	}

	return creds.HasKeys()
}

func ensurePrivateKey(t *testing.T) {
	t.Helper()
	_, err := enosPrivateKey()
	if err != nil {
		t.Skip(fmt.Sprintf("Unable to read acceptance/support/private_key.pem. If you wish to execute this locally you'll need to copy the ENOS_CI_SSH_KEYPAIR-private from 1Password into acceptance/support/private_key.pem: %s", err.Error()))
	}
}

func enosPrivateKey() (string, error) {
	file, err := os.Open("./support/private_key.pem")
	if err != nil {
		return "", err
	}

	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
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

func skipUnlessAWSCredentials() acceptanceRunnerOpt {
	return func(r *acceptanceRunner) {
		r.skipUnlessAWSCredentials = true
	}
}

func skipUnlessEnosPrivateKey() acceptanceRunnerOpt {
	return func(r *acceptanceRunner) {
		r.skipUnlessEnosPrivateKey = true
	}
}

func skipUnlessExtEnabled() acceptanceRunnerOpt {
	return func(r *acceptanceRunner) {
		r.skipUnlessExtEnabled = true
	}
}

// acceptanceRunner is the Enos CLI acceptance test runner
type acceptanceRunner struct {
	enosBinPath              string
	tfBinPath                string
	env                      []string
	skipUnlessTerraformCLI   bool
	skipUnlessAWSCredentials bool
	skipUnlessEnosPrivateKey bool
	skipUnlessExtEnabled     bool
}

// run runs an Enos sub-command
func (r *acceptanceRunner) run(ctx context.Context, subCommand string) ([]byte, error) {
	path, err := filepath.Abs(r.enosBinPath)
	if err != nil {
		return nil, nil
	}
	cmd := exec.CommandContext(ctx, path, strings.Split(subCommand, " ")...)
	cmd.Env = os.Environ()
	return cmd.CombinedOutput()
}

func (r *acceptanceRunner) validate(t *testing.T) {
	t.Helper()
	ensureAcc(t)
	ensureEnosCLI(t)
	if r.skipUnlessTerraformCLI {
		ensureTerraformCLI(t)
	}
	if r.skipUnlessAWSCredentials {
		ensureAWSCredentials(t)
	}
	if r.skipUnlessEnosPrivateKey {
		ensurePrivateKey(t)
	}
	if r.skipUnlessExtEnabled {
		ensureExt(t)
	}
}
