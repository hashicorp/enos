package acceptance

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// TestAcc_Cmd_Scenario_E2E_AWS does an end-to-end test with AWS
func TestAcc_Cmd_Scenario_E2E_AWS(t *testing.T) {
	enos := newAcceptanceRunner(t,
		skipUnlessTerraformCLI(),
		skipUnlessAWSCredentials(),
		skipUnlessEnosPrivateKey(),
		skipUnlessExtEnabled(),
	)

	tmpDir, err := os.MkdirTemp("/tmp", "enos.aws.e2e")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	for _, test := range []struct {
		dir      string
		name     string
		variants map[string][][]string
	}{
		{
			"scenario_e2e_aws",
			"e2e",
			map[string][][]string{
				fmt.Sprintf("%x", sha256.Sum256([]byte("e2e [aws_region:east distro:rhel]"))): {
					{"aws_region", "east"}, {"distro", "rhel"},
				},
				fmt.Sprintf("%x", sha256.Sum256([]byte("e2e [aws_region:west distro:ubuntu]"))): {
					{"aws_region", "west"}, {"distro", "ubuntu"},
				},
			},
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			outDir := filepath.Join(tmpDir, test.dir)
			err := os.MkdirAll(outDir, 0o755)
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
				for uid := range test.variants {
					bytes, err := os.ReadFile(filepath.Join(outDir, uid, "scenario.tf"))
					if err != nil {
						t.Logf("unable to read failed scenario's generated module: %s", err.Error())
					} else {
						t.Logf("%s/scenario.tf\n (%s) %s", uid, test.name, string(bytes))
					}

					bytes, err = os.ReadFile(filepath.Join(outDir, uid, "terraform.rc"))
					if err != nil {
						t.Logf("unable to read failed scenario's generated cli config: %s", err.Error())
					} else {
						t.Logf("%s/terraform.rc (%s)\n %s", uid, test.name, string(bytes))
					}
				}

				// Lets try one more time to destroy resources that might have been
				// created
				out, err := enos.run(context.Background(), fmt.Sprintf("scenario destroy --chdir %s --out %s", path, outDir))
				assert.NoErrorf(t, err, string(out))
			})

			expected := &pb.RunScenariosResponse{
				Responses: []*pb.Scenario_Command_Run_Response{},
			}
			for uid, variants := range test.variants {
				elements := []*pb.Scenario_Filter_Element{}
				for _, variant := range variants {
					elements = append(elements, &pb.Scenario_Filter_Element{
						Key:   variant[0],
						Value: variant[1],
					})
				}

				res := &pb.Scenario_Command_Run_Response{
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
				}

				expected.Responses = append(expected.Responses, res)
			}

			cmd := fmt.Sprintf("scenario run --chdir %s --out %s --format json", path, outDir)
			out, err := enos.run(context.Background(), cmd)
			if err != nil {
				failed = true
			}
			require.NoError(t, err, string(out))

			got := &pb.RunScenariosResponse{}
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
