package acceptance

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// TestAcc_Cmd_Scenario_Destroy tests that a scenario can be generated and validated
// with Terraform.
func TestAcc_Cmd_Scenario_Destroy(t *testing.T) {
	enos := newAcceptanceRunner(t, skipUnlessTerraformCLI())

	tmpDir, err := os.MkdirTemp("/tmp", "enos.destroy")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

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
			outDir := filepath.Join(tmpDir, test.dir)
			err := os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
			outDir, err = filepath.EvalSymlinks(outDir)
			require.NoError(t, err)
			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)

			filter := test.name
			elements := []*pb.Scenario_Filter_Element{}
			for _, variant := range test.variants {
				filter = fmt.Sprintf("%s %s:%s", filter, variant[0], variant[1])
				elements = append(elements, &pb.Scenario_Filter_Element{
					Key:   variant[0],
					Value: variant[1],
				})
			}

			cmd := fmt.Sprintf("scenario launch --chdir %s --out %s %s", path, outDir, filter)
			out, err := enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))

			cmd = fmt.Sprintf("scenario destroy --chdir %s --out %s --format json %s", path, outDir, filter)
			out, err = enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))

			expected := &pb.DestroyScenariosResponse{
				Responses: []*pb.Scenario_Operation_Destroy_Response{
					{
						TerraformModule: &pb.Terraform_Module{
							ModulePath: filepath.Join(outDir, test.uid, "scenario.tf"),
							RcPath:     filepath.Join(outDir, test.uid, "terraform.rc"),
							ScenarioRef: &pb.Ref_Scenario{
								Id: &pb.Scenario_ID{
									Name: test.name,
									Uid:  test.uid,
									Variants: &pb.Scenario_Filter_Vector{
										Elements: elements,
									},
								},
							},
						},
					},
				},
			}

			got := &pb.DestroyScenariosResponse{}
			require.NoErrorf(t, protojson.Unmarshal(out, got), string(out))
			require.Len(t, got.GetResponses(), len(expected.GetResponses()))
			for i := range expected.Responses {
				got := got.Responses[i].TerraformModule
				expected := expected.Responses[i].TerraformModule

				require.Equal(t, expected.ModulePath, got.ModulePath)
				require.Equal(t, expected.RcPath, got.RcPath)
				require.Equal(t, expected.ScenarioRef.String(), got.ScenarioRef.String())
			}
		})
	}
}
