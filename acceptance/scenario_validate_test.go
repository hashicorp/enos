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

// TestAcc_Cmd_Scenario_Validate tests that a scenario can be generated and validated
// with Terraform.
func TestAcc_Cmd_Scenario_Validate(t *testing.T) {
	enos := newAcceptanceRunner(t, skipUnlessTerraformCLI())

	tmpDir, err := os.MkdirTemp("/tmp", "enos.validate")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

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
			outDir := filepath.Join(tmpDir, test.dir)
			err = os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
			outDir, err = filepath.EvalSymlinks(outDir)
			require.NoError(t, err)
			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)

			cmd := fmt.Sprintf("scenario validate --chdir %s --out %s --format json", path, outDir)
			out, err := enos.run(context.Background(), cmd)
			require.NoError(t, err, string(out))

			expected := &pb.ValidateScenariosResponse{
				Responses: []*pb.Scenario_Command_Validate_Response{},
			}

			for _, variant := range test.variants {
				name := test.name
				elements := []*pb.Scenario_Filter_Element{}
				if len(variant) == 2 {
					name = fmt.Sprintf("%s [%s:%s]", name, variant[0], variant[1])
					elements = append(elements, &pb.Scenario_Filter_Element{
						Key:   variant[0],
						Value: variant[1],
					})
				}
				uid := fmt.Sprintf("%x", sha256.Sum256([]byte(name)))

				expected.Responses = append(expected.Responses, &pb.Scenario_Command_Validate_Response{
					Generate: &pb.Scenario_Command_Generate_Response{
						TerraformModule: &pb.Terraform_Module{
							ModulePath: filepath.Join(outDir, uid, "scenario.tf"),
							RcPath:     filepath.Join(outDir, uid, "terraform.rc"),
							ScenarioRef: &pb.Ref_Scenario{
								Id: &pb.Scenario_ID{
									Name: test.name,
									Uid:  uid,
									Variants: &pb.Scenario_Filter_Vector{
										Elements: elements,
									},
								},
							},
						},
					},
					Validate: &pb.Terraform_Command_Validate_Response{
						Valid:         true,
						FormatVersion: "1.0",
					},
				})
			}

			got := &pb.ValidateScenariosResponse{}
			require.NoErrorf(t, protojson.Unmarshal(out, got), string(out))
			require.Len(t, got.GetResponses(), len(expected.GetResponses()))
			for i := range expected.Responses {
				got := got.Responses[i]
				expected := expected.Responses[i]

				require.Equal(t, expected.Generate.TerraformModule.ModulePath, got.Generate.TerraformModule.ModulePath)
				require.Equal(t, expected.Generate.TerraformModule.RcPath, got.Generate.TerraformModule.RcPath)
				require.Equal(t, expected.Generate.TerraformModule.ScenarioRef.String(), got.Generate.TerraformModule.ScenarioRef.String())
				require.Equal(t, expected.Validate.Valid, got.Validate.Valid)
			}
		})
	}
}
