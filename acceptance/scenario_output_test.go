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

// TestAcc_Cmd_Scenario_Output tests that a Terraform output command succeeds
func TestAcc_Cmd_Scenario_Output(t *testing.T) {
	enos := newAcceptanceRunner(t, skipUnlessTerraformCLI())

	tmpDir, err := os.MkdirTemp("/tmp", "enos.exec")
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
			"scenario_generate_step_vars",
			"step_vars",
			[][]string{{"arch", "arm"}, {"distro", "rhel"}},
			fmt.Sprintf("%x", sha256.Sum256([]byte("step_vars [arch:arm distro:rhel]"))),
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

			cmd = fmt.Sprintf(`scenario output --name step_reference_unknown --chdir %s --out %s --format json %s`, path, outDir, filter)
			out, err = enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))

			expected := &pb.OutputScenariosResponse{
				Responses: []*pb.Scenario_Command_Output_Response{
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
						Output: &pb.Terraform_Command_Output_Response{
							Meta: []*pb.Terraform_Command_Output_Response_Meta{
								{
									Name: "step_reference_unknown",
								},
							},
						},
					},
				},
			}

			got := &pb.OutputScenariosResponse{}
			require.NoErrorf(t, protojson.Unmarshal(out, got), string(out))
			require.Len(t, got.GetResponses(), len(expected.GetResponses()))
			for i := range expected.Responses {
				got := got.Responses[i]
				expected := expected.Responses[i]

				require.Equal(t, expected.TerraformModule.ModulePath, got.TerraformModule.ModulePath)
				require.Equal(t, expected.TerraformModule.RcPath, got.TerraformModule.RcPath)
				require.Equal(t, expected.TerraformModule.ScenarioRef.String(), got.TerraformModule.ScenarioRef.String())
				require.Equal(t, expected.Output.Meta[0].Name, got.Output.Meta[0].Name)
				require.NotEmpty(t, got.Output.Meta[0].Type)
				require.NotEmpty(t, got.Output.Meta[0].Value)
			}
		})
	}
}
