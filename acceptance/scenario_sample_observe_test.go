// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/hashicorp/enos/internal/ui/machine"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
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
			dir: "invalid_scenarios/sample_empty_frame",
			filter: &pb.Sample_Filter{
				Sample: &pb.Ref_Sample{
					Id: &pb.Sample_ID{
						Name: "smoke_empty_frame",
					},
				},
				Seed:        1234,
				MaxElements: 1,
				MinElements: 1,
			},
			out: &pb.ObserveSampleResponse{
				Diagnostics: []*pb.Diagnostic{},
				Decode: &pb.DecodeResponse{
					Diagnostics: []*pb.Diagnostic{
						{
							Summary: "no scenarios matched filter criteria: smoke",
						},
						{
							Summary: "failed to decode a sample subset frame: ensure that sample subset refers to a scenario and that all specified subset variants exist in the scenario matrix",
							Detail:  "subset: smoke, variants: [arch:not_a_variant]",
						},
					},
				},
			},
			fail: true,
		},
		{
			dir: "scenarios/sample_observe",
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
									Uid:    "bdc1d6d8a1d8332d0ee5ea09e1cb6ad06d47953f8e225de8dc1f0ec3be5eb6a0",
									Filter: "smoke arch:s390x distro:amz",
									Variants: &pb.Matrix_Vector{
										Elements: []*pb.Matrix_Element{
											{Key: "arch", Value: "s390x"},
											{Key: "distro", Value: "amz"},
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
									Uid:    "3f481b933e5406b3ccafc1a6bbe9fbf18f7f407bbb8546b89dab69345a39dc1b",
									Filter: "upgrade arch:amd64 distro:amz",
									Variants: &pb.Matrix_Vector{
										Elements: []*pb.Matrix_Element{
											{Key: "arch", Value: "amd64"},
											{Key: "distro", Value: "amz"},
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
		t.Run(test.dir, func(t *testing.T) {
			t.Parallel()
			enos := newAcceptanceRunner(t)

			path, err := filepath.Abs(filepath.Join(".", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario sample observe %s --chdir %s --format json --min %d --max %d --seed %d",
				test.filter.GetSample().GetId().GetName(),
				path,
				test.filter.GetMinElements(),
				test.filter.GetMaxElements(),
				test.filter.GetSeed(),
			)
			fmt.Println(path)
			stdout, stderr, err := enos.run(context.Background(), cmd)
			if test.fail {
				require.Error(t, err)

				got := &pb.ObserveSampleResponse{}
				require.NoErrorf(t, protojson.Unmarshal(stdout, got), string(stdout))
				require.Len(t, got.GetDiagnostics(), len(test.out.GetDiagnostics()))
				require.Len(t, got.GetDecode().GetDiagnostics(), len(test.out.GetDecode().GetDiagnostics()))
				for i, d := range test.out.GetDiagnostics() {
					require.Equal(t, got.GetDiagnostics()[i].GetSummary(), d.GetSummary())
				}
				for i, d := range test.out.GetDecode().GetDiagnostics() {
					require.Equal(t, d.GetSummary(), got.GetDecode().GetDiagnostics()[i].GetSummary())
					require.Equal(t, d.GetDetail(), got.GetDecode().GetDiagnostics()[i].GetDetail())
				}
				errMsg := &machine.ErrJSON{}
				require.NoError(t, json.Unmarshal(stderr, errMsg))
				require.Len(t, errMsg.Errors, 1)

				return
			}

			require.NoError(t, err)
			got := &pb.ObserveSampleResponse{}
			require.NoError(t, protojson.Unmarshal(stdout, got))
			require.Len(t, got.GetObservation().GetElements(), len(test.out.GetObservation().GetElements()))
			for i := range test.out.GetObservation().GetElements() {
				require.Equal(t, test.out.GetObservation().GetElements()[i].String(), got.GetObservation().GetElements()[i].String())
			}
		})
	}
}
