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

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

func TestAcc_Cmd_Scenario_Sample_List(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		dir string
		out *pb.ListSamplesResponse
	}{
		{
			dir: "./invalid_scenarios/scenario_list_no_scenarios",
			out: &pb.ListSamplesResponse{},
		},
		{
			dir: "./scenarios/sample_list",
			out: &pb.ListSamplesResponse{
				Samples: []*pb.Ref_Sample{
					{
						Id: &pb.Sample_ID{
							Name: "complex",
						},
					},
					{
						Id: &pb.Sample_ID{
							Name: "minimal",
						},
					},
				},
			},
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			t.Parallel()
			enos := newAcceptanceRunner(t)

			path, err := filepath.Abs(test.dir)
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario sample list --chdir %s --format json", path)
			fmt.Println(path)
			out, _, err := enos.run(context.Background(), cmd)
			require.NoError(t, err)
			got := &pb.ListSamplesResponse{}
			require.NoError(t, protojson.Unmarshal(out, got))
			require.Len(t, got.GetSamples(), len(test.out.GetSamples()))
			for i := range test.out.GetSamples() {
				require.Equal(t, test.out.GetSamples()[i].String(), got.GetSamples()[i].String())
			}
		})
	}
}
