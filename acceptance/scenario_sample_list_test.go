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

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func TestAcc_Cmd_Scenario_Sample_List(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		dir  string
		out  *pb.ListSamplesResponse
		fail bool
	}{
		{
			dir: "scenario_list_pass_0",
			out: &pb.ListSamplesResponse{},
		},
		{
			dir: "sample_list",
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

			path, err := filepath.Abs(filepath.Join("./scenarios", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario sample list --chdir %s --format json", path)
			fmt.Println(path)
			out, err := enos.run(context.Background(), cmd)
			if test.fail {
				require.Error(t, err)

				return
			}

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
