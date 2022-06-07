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

// TestAcc_Cmd_Scenario_Generate tests that a scenario can generate into the
// appropriate terraform module and CLI configuration.
func TestAcc_Cmd_Scenario_Generate(t *testing.T) {
	enos := newAcceptanceRunner(t)

	tmpDir, err := os.MkdirTemp("", "enos.generate.out")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

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

			cmd := fmt.Sprintf("scenario generate --chdir %s --out %s %s --format json", path, outDir, filter)
			out, err := enos.run(context.Background(), cmd)
			require.NoErrorf(t, err, string(out))

			expected := &pb.GenerateScenariosResponse{
				Responses: []*pb.Scenario_Operation_Generate_Response{
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

			got := &pb.GenerateScenariosResponse{}
			require.NoErrorf(t, protojson.Unmarshal(out, got), string(out))
			require.Len(t, got.GetResponses(), len(expected.GetResponses()))
			for i := range expected.Responses {
				got := got.Responses[i].GetTerraformModule()
				expected := expected.Responses[i].GetTerraformModule()

				require.Equal(t, expected.ModulePath, got.ModulePath)
				require.Equal(t, expected.RcPath, got.RcPath)
				require.Equal(t, expected.ScenarioRef.String(), got.ScenarioRef.String())
			}
		})
	}
}
