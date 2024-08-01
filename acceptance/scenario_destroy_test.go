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

// TestAcc_Cmd_Scenario_Destroy tests that a scenario can be generated and validated
// with Terraform.
func TestAcc_Cmd_Scenario_Destroy(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		dir  string
		name string
		// We assume the variants will result in one variant for each test
		variants [][]string
		uid      string
		launch   bool
	}{
		{
			"scenario_generate_pass_0",
			"test",
			[][]string{{"foo", "matrixfoo"}},
			fmt.Sprintf("%x", sha256.Sum256([]byte("test [foo:matrixfoo]"))),
			true,
		},
		{
			"scenario_generate_pass_0",
			"test",
			[][]string{{"foo", "matrixbar"}},
			fmt.Sprintf("%x", sha256.Sum256([]byte("test [foo:matrixbar]"))),
			false,
		},
	} {
		t.Run(fmt.Sprintf("%s %s %s %t", test.dir, test.name, test.variants, test.launch), func(t *testing.T) {
			t.Parallel()
			enos := newAcceptanceRunner(t, skipUnlessTerraformCLI())
			tmpDir, err := os.MkdirTemp("/tmp", "enos.destroy")
			require.NoError(t, err)
			t.Cleanup(func() { os.RemoveAll(tmpDir) })
			outDir := filepath.Join(tmpDir, test.dir)
			err = os.MkdirAll(outDir, 0o755)
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

			// Test destroying a scenario with it launched or not
			if test.launch {
				cmd := fmt.Sprintf("scenario launch --chdir %s --out %s %s", path, outDir, filter)
				out, stderr, err := enos.run(context.Background(), cmd)
				require.NoError(t, err, "enos "+cmd+": "+string(out)+string(stderr))
			}

			cmd := fmt.Sprintf("scenario destroy --chdir %s --out %s --format json %s", path, outDir, filter)
			out, stderr, err := enos.run(context.Background(), cmd)
			require.NoError(t, err, "enos "+cmd+": "+string(out)+string(stderr))

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

			expected := &pb.OperationResponses{
				Responses: []*pb.Operation_Response{
					{
						Op: &pb.Ref_Operation{
							Scenario: scenarioRef,
						},
						Status: pb.Operation_STATUS_COMPLETED,
						Value: &pb.Operation_Response_Destroy_{
							Destroy: &pb.Operation_Response_Destroy{
								Generate: &pb.Operation_Response_Generate{
									TerraformModule: &pb.Terraform_Module{
										ModulePath:  filepath.Join(outDir, test.uid, "scenario.tf"),
										RcPath:      filepath.Join(outDir, test.uid, "terraform.rc"),
										ScenarioRef: scenarioRef,
									},
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
