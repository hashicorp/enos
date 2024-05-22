// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

func TestAcc_Cmd_Scenario_Outline(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		dir  string
		out  *pb.OutlineScenariosResponse
		fail bool
	}{
		{
			dir: "scenarios/scenario_outline",
			out: &pb.OutlineScenariosResponse{
				Outlines: []*pb.Scenario_Outline{
					{
						Scenario: &pb.Ref_Scenario{
							Id: &pb.Scenario_ID{
								Name:        "multiple_verifies",
								Description: "This is a multiline description\nof the upgrade scenario.\n",
							},
						},
						Matrix: &pb.Matrix{
							Vectors: []*pb.Matrix_Vector{
								{Elements: []*pb.Matrix_Element{
									{Key: "arch", Value: "amd64"},
									{Key: "arch", Value: "arm64"},
								}},
								{Elements: []*pb.Matrix_Element{
									{Key: "distro", Value: "ubuntu"},
									{Key: "distro", Value: "rhel"},
								}},
							},
						},
						Steps: []*pb.Scenario_Outline_Step{
							{
								Name:        "test",
								Description: "This is an indented\nmultiline step description.\n",
								Verifies: []*pb.Quality{
									{
										Name:        "inline",
										Description: "an inline quality that isn't reused",
									},
									{
										Name:        "the_data_is_durable",
										Description: "The data is durable\nafter an upgrade.\n",
									},
									{
										Name:        "the_tests_pass",
										Description: "The tests all pass!",
									},
								},
							},
						},
						Verifies: []*pb.Quality{
							{
								Name:        "inline",
								Description: "an inline quality that isn't reused",
							},
							{
								Name:        "the_data_is_durable",
								Description: "The data is durable\nafter an upgrade.\n",
							},
							{
								Name:        "the_tests_pass",
								Description: "The tests all pass!",
							},
						},
					},
					{
						Scenario: &pb.Ref_Scenario{
							Id: &pb.Scenario_ID{
								Name:        "singular_verifies",
								Description: "This is a multiline description\nof the upgrade scenario.\n",
							},
						},
						Matrix: &pb.Matrix{
							Vectors: []*pb.Matrix_Vector{
								{Elements: []*pb.Matrix_Element{
									{Key: "arch", Value: "amd64"},
									{Key: "arch", Value: "arm64"},
								}},
								{Elements: []*pb.Matrix_Element{
									{Key: "distro", Value: "ubuntu"},
									{Key: "distro", Value: "rhel"},
								}},
							},
						},
						Steps: []*pb.Scenario_Outline_Step{
							{
								Name:        "test",
								Description: "This is an indented\nmultiline step description.\n",
								Verifies: []*pb.Quality{
									{
										Name:        "the_tests_pass",
										Description: "The tests all pass!",
									},
								},
							},
						},
						Verifies: []*pb.Quality{
							{
								Name:        "the_tests_pass",
								Description: "The tests all pass!",
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			t.Parallel()
			enos := newAcceptanceRunner(t)

			path, err := filepath.Abs(filepath.Join("./", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario outline --chdir %s --format json", path)
			out, _, err := enos.run(context.Background(), cmd)
			if test.fail {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			got := &pb.OutlineScenariosResponse{}
			require.NoError(t, protojson.Unmarshal(out, got))
			require.Len(t, got.GetOutlines(), len(test.out.GetOutlines()))
			for i := range test.out.GetOutlines() {
				require.EqualValues(t, test.out.GetOutlines()[i].GetScenario().GetId().GetName(), got.GetOutlines()[i].GetScenario().GetId().GetName())
				require.EqualValues(t, test.out.GetOutlines()[i].GetScenario().GetId().GetDescription(), got.GetOutlines()[i].GetScenario().GetId().GetDescription())
				require.EqualValues(t, test.out.GetOutlines()[i].GetScenario().GetId().GetFilter(), got.GetOutlines()[i].GetScenario().GetId().GetFilter())
				require.EqualValues(t, test.out.GetOutlines()[i].GetScenario().GetId().GetUid(), got.GetOutlines()[i].GetScenario().GetId().GetUid())
				require.EqualValues(t, test.out.GetOutlines()[i].GetMatrix().String(), got.GetOutlines()[i].GetMatrix().String())
				require.Len(t, got.GetOutlines()[i].GetVerifies(), len(test.out.GetOutlines()[i].GetVerifies()))
				for q := range test.out.GetOutlines()[i].GetVerifies() {
					require.EqualValues(t, test.out.GetOutlines()[i].GetVerifies()[q].GetName(), got.GetOutlines()[i].GetVerifies()[q].GetName())
					require.EqualValues(t, test.out.GetOutlines()[i].GetVerifies()[q].GetDescription(), got.GetOutlines()[i].GetVerifies()[q].GetDescription())
				}
				for s := range test.out.GetOutlines()[i].GetSteps() {
					require.EqualValues(t, test.out.GetOutlines()[i].GetSteps()[s].GetName(), got.GetOutlines()[i].GetSteps()[s].GetName())
					require.EqualValues(t, test.out.GetOutlines()[i].GetSteps()[s].GetDescription(), got.GetOutlines()[i].GetSteps()[s].GetDescription())
					for q := range test.out.GetOutlines()[i].GetSteps()[s].GetVerifies() {
						require.EqualValues(t, test.out.GetOutlines()[i].GetSteps()[s].GetVerifies()[q].GetName(), got.GetOutlines()[i].GetSteps()[s].GetVerifies()[q].GetName())
						require.EqualValues(t, test.out.GetOutlines()[i].GetSteps()[s].GetVerifies()[q].GetDescription(), got.GetOutlines()[i].GetSteps()[s].GetVerifies()[q].GetDescription())
					}
				}
			}
		})
	}
}
