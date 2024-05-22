// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// TestAcc_Cmd_Scenario_Check tests that a scenario can be checked with Terraform.
func TestAcc_Cmd_Scenario_Check(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		dir  string
		name string
		// We assume the variants will result in one variant for each test
		variants [][]string
	}{
		{
			"scenario_generate_pass_0",
			"test",
			[][]string{{"foo", "matrixbar"}, {"foo", "matrixfoo"}},
		},
		{
			"scenario_generate_complex_module_source",
			"path",
			[][]string{{"skip", "keep"}, {"skip", "skip"}},
		},
	} {
		t.Run(fmt.Sprintf("%s %s %s", test.dir, test.name, test.variants), func(t *testing.T) {
			t.Parallel()

			enos := newAcceptanceRunner(t, skipUnlessTerraformCLI())
			tmpDir, err := os.MkdirTemp("/tmp", "enos.check")
			require.NoError(t, err)
			t.Cleanup(func() { os.RemoveAll(tmpDir) })
			outDir := filepath.Join(tmpDir, test.dir)
			err = os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
			outDir, err = filepath.EvalSymlinks(outDir)
			require.NoError(t, err)
			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)

			cmd := fmt.Sprintf("scenario check --chdir %s --out %s --format json", path, outDir)
			out, _, err := enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))

			expected := &pb.OperationResponses{
				Responses: []*pb.Operation_Response{},
			}

			for _, variant := range test.variants {
				name := test.name
				filter := test.name
				elements := []*pb.Matrix_Element{}
				if len(variant) == 2 {
					name = fmt.Sprintf("%s [%s:%s]", name, variant[0], variant[1])
					filter = fmt.Sprintf("%s %s:%s", test.name, variant[0], variant[1])
					elements = append(elements, &pb.Matrix_Element{
						Key:   variant[0],
						Value: variant[1],
					})
				}
				uid := fmt.Sprintf("%x", sha256.Sum256([]byte(name)))
				scenarioRef := &pb.Ref_Scenario{
					Id: &pb.Scenario_ID{
						Name:   test.name,
						Filter: filter,
						Uid:    uid,
						Variants: &pb.Matrix_Vector{
							Elements: elements,
						},
					},
				}

				expected.Responses = append(expected.GetResponses(), &pb.Operation_Response{
					Op: &pb.Ref_Operation{
						Scenario: scenarioRef,
					},
					Status: pb.Operation_STATUS_COMPLETED,
					Value: &pb.Operation_Response_Check_{
						Check: &pb.Operation_Response_Check{
							Generate: &pb.Operation_Response_Generate{
								TerraformModule: &pb.Terraform_Module{
									ModulePath:  filepath.Join(outDir, uid, "scenario.tf"),
									RcPath:      filepath.Join(outDir, uid, "terraform.rc"),
									ScenarioRef: scenarioRef,
								},
							},
							Validate: &pb.Terraform_Command_Validate_Response{
								Valid:         true,
								FormatVersion: "1.0",
							},
						},
					},
				})
			}

			requireEqualOperationResponses(t, expected, out)
		})
	}
}

// TestAcc_Cmd_Scenario_Check_WithWarnings tests that a scenario can be
// that has warnings can be validated and that the program behaves as it should
// when given the --fail-on-warnings flag.
func TestAcc_Cmd_Scenario_Check_WithWarnings(t *testing.T) {
	t.Parallel()

	enos := newAcceptanceRunner(t,
		skipUnlessTerraformCLI(),
		skipUnlessExtEnabled(), // since we need the random provider
	)

	tmpDir, err := os.MkdirTemp("/tmp", "enos.validate")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	for _, failOnWarnings := range []bool{true, false} {
		t.Run(fmt.Sprintf("fail_on_warnings_%t", failOnWarnings), func(t *testing.T) {
			t.Parallel()
			outDir := filepath.Join(tmpDir, "scenario_generate_has_warnings")
			err := os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
			outDir, err = filepath.EvalSymlinks(outDir)
			require.NoError(t, err)
			path, err := filepath.Abs(filepath.Join("./scenarios", "scenario_generate_has_warnings"))
			require.NoError(t, err)

			cmd := fmt.Sprintf("scenario check --chdir %s --out %s --format json", path, outDir)
			if failOnWarnings {
				cmd = cmd + " --fail-on-warnings"
			}
			out, _, err := enos.run(context.Background(), cmd)
			if failOnWarnings {
				require.Error(t, err, string(out))

				return
			}
			require.NoError(t, err, string(out))

			expected := &pb.OperationResponses{
				Responses: []*pb.Operation_Response{},
			}

			for _, variant := range []string{"has_warning", "valid"} {
				name := fmt.Sprintf("warning [mod:%s]", variant)
				elements := []*pb.Matrix_Element{}
				elements = append(elements, &pb.Matrix_Element{
					Key:   "mod",
					Value: variant,
				})
				uid := fmt.Sprintf("%x", sha256.Sum256([]byte(name)))
				scenarioRef := &pb.Ref_Scenario{
					Id: &pb.Scenario_ID{
						Name:   "warning",
						Uid:    uid,
						Filter: "warning mod:" + variant,
						Variants: &pb.Matrix_Vector{
							Elements: elements,
						},
					},
				}
				var diags []*pb.Diagnostic
				warningCount := 0
				status := pb.Operation_STATUS_COMPLETED
				if variant == "has_warning" {
					diags = append(diags, &pb.Diagnostic{
						// We don't need to include details since
						// our validation only looks at diags count
						Severity: pb.Diagnostic_SEVERITY_WARNING,
					})
					warningCount = 1

					if failOnWarnings {
						status = pb.Operation_STATUS_FAILED
					} else {
						status = pb.Operation_STATUS_COMPLETED_WARNING
					}
				}

				expected.Responses = append(expected.GetResponses(), &pb.Operation_Response{
					Op: &pb.Ref_Operation{
						Scenario: scenarioRef,
					},
					Status: status,
					Value: &pb.Operation_Response_Check_{
						Check: &pb.Operation_Response_Check{
							Generate: &pb.Operation_Response_Generate{
								TerraformModule: &pb.Terraform_Module{
									ModulePath:  filepath.Join(outDir, uid, "scenario.tf"),
									RcPath:      filepath.Join(outDir, uid, "terraform.rc"),
									ScenarioRef: scenarioRef,
								},
							},
							Validate: &pb.Terraform_Command_Validate_Response{
								Diagnostics:   diags,
								Valid:         true,
								FormatVersion: "1.0",
								WarningCount:  int64(warningCount),
							},
							Plan: &pb.Terraform_Command_Plan_Response{
								ChangesPresent: true,
							},
						},
					},
				})
			}

			requireEqualOperationResponses(t, expected, out)
		})
	}
}
