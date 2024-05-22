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

// TestAcc_Cmd_Scenario_Generate tests that a scenario can generate into the
// appropriate terraform module and CLI configuration.
func TestAcc_Cmd_Scenario_Generate(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		dir  string
		name string
		// We assume the variants will result in one generated module
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
		{
			"scenario_generate_pass_backend",
			"test",
			[][]string{},
			fmt.Sprintf("%x", sha256.Sum256([]byte("test"))),
		},
		{
			"scenario_generate_pass_cloud",
			"test",
			[][]string{},
			fmt.Sprintf("%x", sha256.Sum256([]byte("test"))),
		},
		{
			"scenario_generate_step_vars",
			"step_vars",
			[][]string{{"arch", "arm"}, {"distro", "rhel"}},
			fmt.Sprintf("%x", sha256.Sum256([]byte("step_vars [arch:arm distro:rhel]"))),
		},
		{
			"scenario_generate_complex_module_source",
			"path",
			[][]string{{"skip", "keep"}},
			fmt.Sprintf("%x", sha256.Sum256([]byte("path [skip:keep]"))),
		},
		{
			"scenario_generate_complex_provider",
			"kubernetes",
			[][]string{},
			fmt.Sprintf("%x", sha256.Sum256([]byte("kubernetes"))),
		},
	} {
		t.Run(fmt.Sprintf("%s %s %s", test.dir, test.name, test.variants), func(t *testing.T) {
			t.Parallel()

			enos := newAcceptanceRunner(t)

			tmpDir, err := os.MkdirTemp("", "enos.generate.out")
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

			var variants *pb.Matrix_Vector
			if len(elements) > 0 {
				variants = &pb.Matrix_Vector{
					Elements: elements,
				}
			}

			scenarioRef := &pb.Ref_Scenario{
				Id: &pb.Scenario_ID{
					Name:     test.name,
					Filter:   filter,
					Uid:      test.uid,
					Variants: variants,
				},
			}

			cmd := fmt.Sprintf("scenario generate --chdir %s --out %s %s --format json", path, outDir, filter)
			out, _, err := enos.run(context.Background(), cmd)
			require.NoErrorf(t, err, string(out))

			expected := &pb.OperationResponses{
				Responses: []*pb.Operation_Response{
					{
						Op: &pb.Ref_Operation{
							Scenario: scenarioRef,
						},
						Status: pb.Operation_STATUS_COMPLETED,
						Value: &pb.Operation_Response_Generate_{
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
			}

			requireEqualOperationResponses(t, expected, out)
		})
	}
}
