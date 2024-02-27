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
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func TestAcc_Cmd_Scenario_Sample_Observe(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		dir    string
		sample string
		filter *pb.Sample_Filter
		out    *pb.ObserveSampleResponse
		fail   bool
	}{
		{
			dir: "sample_observe",
			filter: &pb.Sample_Filter{
				Sample: &pb.Ref_Sample{
					Id: &pb.Sample_ID{
						Name: "all",
					},
				},
				Seed:        1234,
				MaxElements: 3,
				MinElements: 3,
			},
			out: &pb.ObserveSampleResponse{
				Observation: &pb.Sample_Observation{
					Elements: []*pb.Sample_Element{
						{
							Sample: &pb.Ref_Sample{
								Id: &pb.Sample_ID{
									Name: "all",
								},
							},
							Subset: &pb.Ref_Sample_Subset{
								Id: &pb.Sample_Subset_ID{
									Name: "smoke",
								},
							},
							Scenario: &pb.Ref_Scenario{
								Id: &pb.Scenario_ID{
									Name:   "smoke",
									Uid:    "ebc8083d61b560ea1678efb642cfdf54034a7374e18d27e757e71bb5dc28c62e",
									Filter: "smoke arch:arm64 distro:rhel",
									Variants: &pb.Matrix_Vector{
										Elements: []*pb.Matrix_Element{
											{Key: "arch", Value: "arm64"},
											{Key: "distro", Value: "rhel"},
										},
									},
								},
							},
							Attributes: &structpb.Struct{
								Fields: map[string]*structpb.Value{
									"aws-region":        structpb.NewStringValue("us-east-1"),
									"continue-on-error": structpb.NewBoolValue(false),
									"notify-on-fail":    structpb.NewBoolValue(true),
								},
							},
						},
						{
							Sample: &pb.Ref_Sample{
								Id: &pb.Sample_ID{
									Name: "all",
								},
							},
							Subset: &pb.Ref_Sample_Subset{
								Id: &pb.Sample_Subset_ID{
									Name: "smoke_allow_failure",
								},
							},
							Scenario: &pb.Ref_Scenario{
								Id: &pb.Scenario_ID{
									Name:   "smoke",
									Uid:    "21610357b0083126896a0072a429e677fd6381fa98e94b15ac2101f697100b42",
									Filter: "smoke arch:s390x distro:ubuntu",
									Variants: &pb.Matrix_Vector{
										Elements: []*pb.Matrix_Element{
											{Key: "arch", Value: "s390x"},
											{Key: "distro", Value: "ubuntu"},
										},
									},
								},
							},
							Attributes: &structpb.Struct{
								Fields: map[string]*structpb.Value{
									"aws-region":        structpb.NewStringValue("us-east-1"),
									"continue-on-error": structpb.NewBoolValue(true),
									"notify-on-fail":    structpb.NewBoolValue(true),
								},
							},
						},
						{
							Sample: &pb.Ref_Sample{
								Id: &pb.Sample_ID{
									Name: "all",
								},
							},
							Subset: &pb.Ref_Sample_Subset{
								Id: &pb.Sample_Subset_ID{
									Name: "upgrade",
								},
							},
							Scenario: &pb.Ref_Scenario{
								Id: &pb.Scenario_ID{
									Name:   "upgrade",
									Uid:    "f662b9deac220ad5bc22f2618098925b47d06fc342d8c0d0e6240bb67547b9e8",
									Filter: "upgrade arch:arm64 distro:ubuntu",
									Variants: &pb.Matrix_Vector{
										Elements: []*pb.Matrix_Element{
											{Key: "arch", Value: "arm64"},
											{Key: "distro", Value: "ubuntu"},
										},
									},
								},
							},
							Attributes: &structpb.Struct{
								Fields: map[string]*structpb.Value{
									"aws-region":        structpb.NewStringValue("us-west-1"),
									"continue-on-error": structpb.NewBoolValue(false),
								},
							},
						},
					},
				},
			},
		},
	} {
		test := test
		t.Run(test.dir, func(t *testing.T) {
			t.Parallel()
			enos := newAcceptanceRunner(t)

			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario sample observe %s --chdir %s --format json --min %d --max %d --seed %d",
				test.filter.GetSample().GetId().GetName(),
				path,
				test.filter.GetMinElements(),
				test.filter.GetMaxElements(),
				test.filter.GetSeed(),
			)
			fmt.Println(path)
			out, err := enos.run(context.Background(), cmd)
			if test.fail {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			got := &pb.ObserveSampleResponse{}
			require.NoError(t, protojson.Unmarshal(out, got))
			require.Len(t, got.GetObservation().GetElements(), len(test.out.GetObservation().GetElements()))
			for i := range test.out.GetObservation().GetElements() {
				require.Equal(t, test.out.GetObservation().GetElements()[i].String(), got.GetObservation().GetElements()[i].String())
			}
		})
	}
}
