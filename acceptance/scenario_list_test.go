package acceptance

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func TestAcc_Cmd_Scenario_List(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		dir  string
		out  *pb.ListScenariosResponse
		fail bool
	}{
		{
			dir: "scenario_list_pass_0",
			out: &pb.ListScenariosResponse{},
		},
		{
			dir: "scenario_list_pass_1",
			out: &pb.ListScenariosResponse{
				Scenarios: []*pb.Ref_Scenario{{
					Id: &pb.Scenario_ID{
						Name:     "test",
						Uid:      "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
						Variants: &pb.Scenario_Filter_Vector{},
					},
				}},
			},
		},
		{
			dir: "scenario_list_pass_2",
			out: &pb.ListScenariosResponse{
				Scenarios: []*pb.Ref_Scenario{
					{
						Id: &pb.Scenario_ID{
							Name:     "consul",
							Uid:      "b713f0bd8f48dfad2263cabc455ade78f7e4e99a548101f31f935686dff67124",
							Variants: &pb.Scenario_Filter_Vector{},
						},
					},
					{
						Id: &pb.Scenario_ID{
							Name:     "vault",
							Uid:      "e6f0a1fbb43c89196dcfcbef85908f19ab4c5f7cc4f4c452284697757683d7ef",
							Variants: &pb.Scenario_Filter_Vector{},
						},
					},
				},
			},
		},
		{
			dir: "scenario_list_pass_3",
			out: &pb.ListScenariosResponse{
				Scenarios: []*pb.Ref_Scenario{
					{
						Id: &pb.Scenario_ID{
							Name: "test",
							Uid:  "5ee261842ccc5bece062285d63a36dafc61bb5b95793f55820a885969ab8b19b",
							Variants: &pb.Scenario_Filter_Vector{
								Elements: []*pb.Scenario_Filter_Element{
									{
										Key:   "backend",
										Value: "consul",
									},
								},
							},
						},
					},
					{
						Id: &pb.Scenario_ID{
							Name: "test",
							Uid:  "c3576214aca53aad678161d049f5c123026bff0fb5ec1761438c32114fe445a0",
							Variants: &pb.Scenario_Filter_Vector{
								Elements: []*pb.Scenario_Filter_Element{
									{
										Key:   "backend",
										Value: "raft",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			dir:  "scenario_list_fail_malformed",
			fail: true,
		},
	} {
		test := test
		t.Run(test.dir, func(t *testing.T) {
			t.Parallel()
			enos := newAcceptanceRunner(t)

			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario list --chdir %s --format json", path)
			fmt.Println(path)
			out, err := enos.run(context.Background(), cmd)
			if test.fail {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			got := &pb.ListScenariosResponse{}
			require.NoError(t, protojson.Unmarshal(out, got))
			require.Len(t, got.GetScenarios(), len(test.out.GetScenarios()))
			for i := range test.out.Scenarios {
				require.Equal(t, test.out.Scenarios[i].String(), got.Scenarios[i].String())
			}
		})
	}
}
