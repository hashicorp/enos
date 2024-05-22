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

	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// TestAcc_Cmd_Scenario_E2E_AWS does an end-to-end test with AWS.
func TestAcc_Cmd_Scenario_E2E_AWS(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		dir      string
		name     string
		variants []struct {
			uid      string
			filter   string
			variants [][]string
		}
	}{
		{
			dir:  "scenario_e2e_aws",
			name: "e2e",
			variants: []struct {
				uid      string
				filter   string
				variants [][]string
			}{
				{
					fmt.Sprintf("%x", sha256.Sum256([]byte("e2e [aws_region:east distro:rhel]"))),
					"e2e aws_region:east distro:rhel",
					[][]string{{"aws_region", "east"}, {"distro", "rhel"}},
				},
				{
					fmt.Sprintf("%x", sha256.Sum256([]byte("e2e [aws_region:west distro:ubuntu]"))),
					"e2e aws_region:west distro:ubuntu",
					[][]string{{"aws_region", "west"}, {"distro", "ubuntu"}},
				},
			},
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			t.Parallel()

			enos := newAcceptanceRunner(t,
				skipUnlessTerraformCLI(),
				skipUnlessAWSCredentials(),
				skipUnlessEnosPrivateKey(),
				skipUnlessExtEnabled(),
			)

			tmpDir, err := os.MkdirTemp("/tmp", "enos.aws.e2e")
			require.NoError(t, err)
			t.Cleanup(func() { os.RemoveAll(tmpDir) })
			outDir := filepath.Join(tmpDir, test.dir)
			err = os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
			outDir, err = filepath.EvalSymlinks(outDir)
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
				for _, variant := range test.variants {
					bytes, err := os.ReadFile(filepath.Join(outDir, variant.uid, "scenario.tf"))
					if err != nil {
						t.Logf("unable to read failed scenario's generated module: %s", err.Error())
					} else {
						t.Logf("%s/scenario.tf\n (%s) %s", variant.uid, test.name, string(bytes))
					}

					bytes, err = os.ReadFile(filepath.Join(outDir, variant.uid, "terraform.rc"))
					if err != nil {
						t.Logf("unable to read failed scenario's generated cli config: %s", err.Error())
					} else {
						t.Logf("%s/terraform.rc (%s)\n %s", variant.uid, test.name, string(bytes))
					}
				}

				// Lets try one more time to destroy resources that might have been
				// created
				out, _, err := enos.run(context.Background(), fmt.Sprintf("scenario destroy --chdir %s --out %s", path, outDir))
				require.NoErrorf(t, err, string(out))
			})

			expected := &pb.OperationResponses{
				Responses: []*pb.Operation_Response{},
			}
			for _, variant := range test.variants {
				elements := []*pb.Matrix_Element{}
				for _, v := range variant.variants {
					elements = append(elements, &pb.Matrix_Element{
						Key:   v[0],
						Value: v[1],
					})
				}

				scenarioRef := &pb.Ref_Scenario{
					Id: &pb.Scenario_ID{
						Name:   test.name,
						Uid:    variant.uid,
						Filter: variant.filter,
						Variants: &pb.Matrix_Vector{
							Elements: elements,
						},
					},
				}

				res := &pb.Operation_Response{
					Op: &pb.Ref_Operation{
						Scenario: scenarioRef,
					},
					Status: pb.Operation_STATUS_COMPLETED,
					Value: &pb.Operation_Response_Run_{
						Run: &pb.Operation_Response_Run{
							Generate: &pb.Operation_Response_Generate{
								TerraformModule: &pb.Terraform_Module{
									ModulePath:  filepath.Join(outDir, variant.uid, "scenario.tf"),
									RcPath:      filepath.Join(outDir, variant.uid, "terraform.rc"),
									ScenarioRef: scenarioRef,
								},
							},
							Validate: &pb.Terraform_Command_Validate_Response{
								Valid:         true,
								FormatVersion: "1.0",
							},
							Plan: &pb.Terraform_Command_Plan_Response{
								ChangesPresent: true,
							},
						},
					},
				}

				expected.Responses = append(expected.GetResponses(), res)
			}

			cmd := fmt.Sprintf("scenario run --chdir %s --out %s --format json", path, outDir)
			out, _, err := enos.run(context.Background(), cmd)
			if err != nil {
				failed = true
			}
			require.NoError(t, err, string(out))

			requireEqualOperationResponses(t, expected, out)
		})
	}
}
