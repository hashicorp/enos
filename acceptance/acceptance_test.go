// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
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

	if cfg.Credentials == nil {
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
		t.Skipf("Unable to read acceptance/support/private_key.pem. If you wish to execute this locally you'll need to copy the ENOS_CI_SSH_KEYPAIR-private from 1Password into acceptance/support/private_key.pem: %s", err.Error())
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

// acceptanceRunner is the Enos CLI acceptance test runner.
type acceptanceRunner struct {
	enosBinPath              string
	tfBinPath                string
	env                      []string
	skipUnlessTerraformCLI   bool
	skipUnlessAWSCredentials bool
	skipUnlessEnosPrivateKey bool
	skipUnlessExtEnabled     bool
}

// run runs an Enos sub-command.
func (r *acceptanceRunner) run(ctx context.Context, subCommand string) ([]byte, []byte, error) {
	path, err := filepath.Abs(r.enosBinPath)
	if err != nil {
		return nil, nil, err
	}

	cmdParts := strings.Split(subCommand, " ")
	// Don't specify a port so we can execute tests in parallel
	cmdParts = append(cmdParts, "--grpc-listen", "http://localhost")

	cmd := exec.CommandContext(ctx, path, cmdParts...)
	cmd.Env = os.Environ()

	stdout, err := cmd.Output()
	var stderr []byte
	var exitErr *exec.ExitError
	if err != nil && errors.As(err, &exitErr) {
		stderr = exitErr.Stderr
	}

	return stdout, stderr, err
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

func sortResponses(r []*pb.Operation_Response) {
	sort.Slice(r, func(i, j int) bool {
		is := flightplan.NewScenario()
		is.FromRef(r[i].GetOp().GetScenario())

		js := flightplan.NewScenario()
		js.FromRef(r[j].GetOp().GetScenario())

		return is.String() < js.String()
	})
}

func requireEqualOperationResponses(t *testing.T, expected *pb.OperationResponses, out []byte) {
	t.Helper()

	got := &pb.OperationResponses{}
	require.NoErrorf(t, protojson.Unmarshal(out, got), string(out))
	require.Len(t, expected.GetResponses(), len(got.GetResponses()))
	expectedResponses := expected.GetResponses()
	gotResponses := got.GetResponses()
	sortResponses(expectedResponses)
	sortResponses(gotResponses)

	require.Lenf(t, gotResponses, len(expectedResponses),
		fmt.Sprintf("expected %d responses, got %d", len(expectedResponses), len(gotResponses)),
	)
	for i := range expectedResponses {
		require.NotNil(t, gotResponses)
		expected := expectedResponses[i]
		got := gotResponses[i]

		// Scenario reference
		require.Equal(t, expected.GetOp().GetScenario().String(), got.GetOp().GetScenario().String())

		// Status
		require.Equalf(t,
			expected.GetStatus(), got.GetStatus(),
			"expected status %s, got %s",
			pb.Operation_Status_name[int32(expected.GetStatus())],
			pb.Operation_Status_name[int32(got.GetStatus())],
		)

		// Generate response type
		requireEqualGenerateResponse(t, expected.GetGenerate(), got.GetGenerate())

		// Check response type
		requireEqualGenerateResponse(t, expected.GetCheck().GetGenerate(), got.GetCheck().GetGenerate())
		requireEqualInitReponse(t, expected.GetCheck().GetInit(), got.GetCheck().GetInit())
		requireEqualValidate(t, expected.GetCheck().GetValidate(), got.GetCheck().GetValidate())
		requireEqualPlan(t, expected.GetCheck().GetPlan(), got.GetCheck().GetPlan())

		// Launch response type
		requireEqualGenerateResponse(t, expected.GetLaunch().GetGenerate(), got.GetLaunch().GetGenerate())
		requireEqualInitReponse(t, expected.GetLaunch().GetInit(), got.GetLaunch().GetInit())
		requireEqualValidate(t, expected.GetLaunch().GetValidate(), got.GetLaunch().GetValidate())
		requireEqualPlan(t, expected.GetLaunch().GetPlan(), got.GetLaunch().GetPlan())
		requireEqualApply(t, expected.GetLaunch().GetApply(), got.GetLaunch().GetApply())

		// Destroy response type
		requireEqualGenerateResponse(t, expected.GetDestroy().GetGenerate(), got.GetDestroy().GetGenerate())
		requireEqualDestroy(t, expected.GetDestroy().GetDestroy(), got.GetDestroy().GetDestroy())

		// Run response type
		requireEqualGenerateResponse(t, expected.GetRun().GetGenerate(), got.GetRun().GetGenerate())
		requireEqualInitReponse(t, expected.GetRun().GetInit(), got.GetRun().GetInit())
		requireEqualValidate(t, expected.GetRun().GetValidate(), got.GetRun().GetValidate())
		requireEqualPlan(t, expected.GetRun().GetPlan(), got.GetRun().GetPlan())
		requireEqualApply(t, expected.GetRun().GetApply(), got.GetRun().GetApply())
		requireEqualDestroy(t, expected.GetRun().GetDestroy(), got.GetRun().GetDestroy())

		// Output response type
		requireEqualOutput(t, expected.GetOutput().GetOutput(), got.GetOutput().GetOutput())

		// Exec response type
		requireEqualExec(t, expected.GetExec().GetExec(), got.GetExec().GetExec())
	}
}

func requireEqualGenerateResponse(t *testing.T, expected, got *pb.Operation_Response_Generate) {
	t.Helper()

	if expected.GetTerraformModule().GetModulePath() != "" {
		require.Equal(t, expected.GetTerraformModule().GetModulePath(),
			got.GetTerraformModule().GetModulePath(),
		)
	}
	if expected.GetTerraformModule().GetRcPath() != "" {
		require.Equal(t, expected.GetTerraformModule().GetRcPath(),
			got.GetTerraformModule().GetRcPath(),
		)
	}
	require.Equal(t, expected.GetTerraformModule().GetScenarioRef().String(),
		got.GetTerraformModule().GetScenarioRef().String(),
	)
}

func requireEqualInitReponse(t *testing.T, expected, got *pb.Terraform_Command_Init_Response) {
	t.Helper()

	require.Equal(t, expected.GetStderr(), got.GetStderr())
	require.Len(t, expected.GetDiagnostics(), len(got.GetDiagnostics()))
}

func requireEqualValidate(t *testing.T, expected, got *pb.Terraform_Command_Validate_Response) {
	t.Helper()

	require.Equal(t, expected.GetValid(), got.GetValid())
	require.Equal(t, expected.GetWarningCount(), got.GetWarningCount())
	require.Len(t, expected.GetDiagnostics(), len(got.GetDiagnostics()))
}

func requireEqualPlan(t *testing.T, expected, got *pb.Terraform_Command_Plan_Response) {
	t.Helper()

	require.Equal(t, expected.GetChangesPresent(), got.GetChangesPresent())
	require.Equal(t, expected.GetStderr(), got.GetStderr())
	require.Len(t, expected.GetDiagnostics(), len(got.GetDiagnostics()))
}

func requireEqualApply(t *testing.T, expected, got *pb.Terraform_Command_Apply_Response) {
	t.Helper()

	require.Equal(t, expected.GetStderr(), got.GetStderr())
	require.Len(t, expected.GetDiagnostics(), len(got.GetDiagnostics()))
}

func requireEqualDestroy(t *testing.T, expected, got *pb.Terraform_Command_Destroy_Response) {
	t.Helper()

	require.Equal(t, expected.GetStderr(), got.GetStderr())
	require.Len(t, expected.GetDiagnostics(), len(got.GetDiagnostics()))
}

func requireEqualOutput(t *testing.T, expected, got *pb.Terraform_Command_Output_Response) {
	t.Helper()

	require.Len(t, expected.GetMeta(), len(got.GetMeta()))
	for i, eMeta := range expected.GetMeta() {
		gotMetas := got.GetMeta()
		require.NotNil(t, gotMetas)
		gotMeta := gotMetas[i]
		require.NotNil(t, gotMeta)

		require.Equal(t, eMeta.GetName(), gotMeta.GetName())
		// Skip the type and the value by default since they're encoded
		// require.Equal(t, eMeta.GetType(), gotMeta.GetType())
		// require.Equal(t, eMeta.GetValue(), gotMeta.GetValue())
		require.Equal(t, eMeta.GetSensitive(), gotMeta.GetSensitive())
		require.Equal(t, eMeta.GetStderr(), gotMeta.GetStderr())
	}
	require.Equal(t, expected.GetDiagnostics(), got.GetDiagnostics())
}

func requireEqualExec(t *testing.T, expected, got *pb.Terraform_Command_Exec_Response) {
	t.Helper()

	require.Equal(t, expected.GetSubCommand(), got.GetSubCommand())
	require.Len(t, expected.GetDiagnostics(), len(got.GetDiagnostics()))
	// NOTE: we don't check stderr since anything we could test would be brittle
}
