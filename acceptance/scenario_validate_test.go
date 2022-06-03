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

// TestAcc_Cmd_Scenario_Validate_WithWarnings tests that a scenario can be
// that has warnings can be validated and that the program behaves as it should
// when given the --fail-on-warnings flag.
func TestAcc_Cmd_Scenario_Validate_WithWarnings(t *testing.T) {
	enos := newAcceptanceRunner(t,
		skipUnlessTerraformCLI(),
		skipUnlessExtEnabled(), // since we need the random provider
	)

	tmpDir, err := os.MkdirTemp("/tmp", "enos.validate")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	for _, failOnWarnings := range []bool{true, false} {
		t.Run(fmt.Sprintf("fail_on_warnings_%t", failOnWarnings), func(t *testing.T) {
			outDir := filepath.Join(tmpDir, "scenario_generate_has_warnings")
			err = os.MkdirAll(outDir, 0o755)
			require.NoError(t, err)
			outDir, err = filepath.EvalSymlinks(outDir)
			require.NoError(t, err)
			path, err := filepath.Abs(filepath.Join("./scenarios", "scenario_generate_has_warnings"))
			require.NoError(t, err)

			cmd := fmt.Sprintf("scenario validate --chdir %s --out %s --format json", path, outDir)
			if failOnWarnings {
				cmd = fmt.Sprintf("%s --fail-on-warnings", cmd)
			}
			out, err := enos.run(context.Background(), cmd)
			if failOnWarnings {
				require.Error(t, err, string(out))
				return
			}
			require.NoError(t, err, string(out))

			expected := &pb.ValidateScenariosResponse{
				Responses: []*pb.Scenario_Command_Validate_Response{},
			}

			for _, variant := range []string{"has_warning", "valid"} {
				name := fmt.Sprintf("warning [mod:%s]", variant)
				elements := []*pb.Scenario_Filter_Element{}
				elements = append(elements, &pb.Scenario_Filter_Element{
					Key:   "mod",
					Value: variant,
				})
				uid := fmt.Sprintf("%x", sha256.Sum256([]byte(name)))

				expected.Responses = append(expected.Responses, &pb.Scenario_Command_Validate_Response{
					Generate: &pb.Scenario_Command_Generate_Response{
						TerraformModule: &pb.Terraform_Module{
							ModulePath: filepath.Join(outDir, uid, "scenario.tf"),
							RcPath:     filepath.Join(outDir, uid, "terraform.rc"),
							ScenarioRef: &pb.Ref_Scenario{
								Id: &pb.Scenario_ID{
									Name: "warning",
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
						WarningCount:  1,
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
