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

// TestAcc_Cmd_Scenario_Launch tests that a scenario can be generated and validated
// with Terraform.
func TestAcc_Cmd_Scenario_Launch(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		dir  string
		name string
		// We assume the variants will result in one variant for each test
		variants [][]string
		uid      string
	}{
		{
			"scenario_generate_pass_0",
			"test",
			[][]string{{"foo", "matrixfoo"}},
			fmt.Sprintf("%x", sha256.Sum256([]byte("test [foo:matrixfoo]"))),
		},
		{
			"scenario_generate_pass_0",
			"test",
			[][]string{{"foo", "matrixbar"}},
			fmt.Sprintf("%x", sha256.Sum256([]byte("test [foo:matrixbar]"))),
		},
	} {
		t.Run(fmt.Sprintf("%s %s %s", test.dir, test.name, test.variants), func(t *testing.T) {
			t.Parallel()

			enos := newAcceptanceRunner(t, skipUnlessTerraformCLI())

			tmpDir := t.TempDir()
			outDir := filepath.Join(tmpDir, test.dir)
			err := os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
			outDir, err = filepath.EvalSymlinks(outDir)
			require.NoError(t, err)
			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)

			filter := test.name
			elements := []*pb.Matrix_Element{}
			for _, variant := range test.variants {
				filter = fmt.Sprintf("%s %s:%s", filter, variant[0], variant[1])
				elements = append(elements, &pb.Matrix_Element{
					Key:   variant[0],
					Value: variant[1],
				})
			}

			scenarioRef := &pb.Ref_Scenario{
				Id: &pb.Scenario_ID{
					Name:   test.name,
					Filter: filter,
					Uid:    test.uid,
					Variants: &pb.Matrix_Vector{
						Elements: elements,
					},
				},
			}

			cmd := fmt.Sprintf("scenario launch --chdir %s --out %s --format json %s", path, outDir, filter)
			out, _, err := enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))

			expected := &pb.OperationResponses{
				Responses: []*pb.Operation_Response{
					{
						Op: &pb.Ref_Operation{
							Scenario: scenarioRef,
						},
						Status: pb.Operation_STATUS_COMPLETED,
						Value: &pb.Operation_Response_Launch_{
							Launch: &pb.Operation_Response_Launch{
								Generate: &pb.Operation_Response_Generate{
									TerraformModule: &pb.Terraform_Module{
										ModulePath:  filepath.Join(outDir, test.uid, "scenario.tf"),
										RcPath:      filepath.Join(outDir, test.uid, "terraform.rc"),
										ScenarioRef: scenarioRef,
									},
								},
								Validate: &pb.Terraform_Command_Validate_Response{
									Valid:         true,
									FormatVersion: "1.0",
								},
								Plan: &pb.Terraform_Command_Plan_Response{
									ChangesPresent: false,
								},
							},
						},
					},
				},
			}

			requireEqualOperationResponses(t, expected, out)
		})
	}
}
