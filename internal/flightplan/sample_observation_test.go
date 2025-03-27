// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

func Test_NewSampleObservationReq(t *testing.T) {
	t.Parallel()

	modulePath, err := filepath.Abs("./tests/simple_module")
	require.NoError(t, err)

	body := fmt.Sprintf(`
module "foo" {
  source = "%s"
}

scenario "foo" {
  matrix {
    length = ["fl1", "fl2", "fl3"]
    width  = ["fw1", "fw2", "fw3"]
  }

  step "foo" {
    module = module.foo
  }
}

scenario "bar" {
  matrix {
    length = ["bl1", "bl2", "bl3"]
    width  = ["bw1", "bw2", "bw3"]
  }

  step "foo" {
    module = module.foo
  }
}

scenario "simple" {
  step "foo" {
    module = module.foo
  }
}

sample "simple" {
  subset "simple" { }
}

sample "foo" {
  subset "foo" {
    matrix {
      length = ["fl2", "fl3"]
      width  = ["fw1", "fw3"]
    }

    attributes = {
      foo = "bar"
      hello = ["ohai", "howdy"]
    }
  }

  subset "simple" { }
}`, modulePath)

	for desc, test := range map[string]struct {
		opts       []SampleObservationOpt
		expected   *SampleObservationReq
		shouldFail bool
	}{
		"no workspace": {
			opts: []SampleObservationOpt{
				WithSampleObservationReqFunc(SampleFuncAll),
				WithSampleObservationReqFilter(&pb.Sample_Filter{
					Sample: &pb.Ref_Sample{
						Id: &pb.Sample_ID{
							Name: "foo",
						},
					},
					Subsets: []*pb.Sample_Subset_ID{
						{Name: "foo"},
					},
				}),
			},
			shouldFail: true,
		},
		"no filter": {
			opts: []SampleObservationOpt{
				WithSampleObservationReqFunc(SampleFuncAll),
				WithSampleObservationReqWorkSpace(testCreateWireWorkspace(t, withTestCreateWireWorkspaceBody(body))),
			},
			shouldFail: true,
		},
		"no sample name in filter": {
			opts: []SampleObservationOpt{
				WithSampleObservationReqFunc(SampleFuncAll),
				WithSampleObservationReqWorkSpace(testCreateWireWorkspace(t, withTestCreateWireWorkspaceBody(body))),
				WithSampleObservationReqFilter(&pb.Sample_Filter{
					Sample: &pb.Ref_Sample{
						Id: &pb.Sample_ID{},
					},
					Subsets: []*pb.Sample_Subset_ID{
						{Name: "foo"},
					},
				}),
			},
			shouldFail: true,
		},
		"no flightplan": {
			opts: []SampleObservationOpt{
				WithSampleObservationReqFunc(SampleFuncAll),
				WithSampleObservationReqWorkSpace(&pb.Workspace{}),
				WithSampleObservationReqFilter(&pb.Sample_Filter{
					Sample: &pb.Ref_Sample{
						Id: &pb.Sample_ID{
							Name: "foo",
						},
					},
					Subsets: []*pb.Sample_Subset_ID{
						{Name: "foo"},
					},
				}),
			},
			shouldFail: true,
		},
		"valid": {
			opts: []SampleObservationOpt{
				WithSampleObservationReqFunc(SampleFuncAll),
				WithSampleObservationReqWorkSpace(testCreateWireWorkspace(t, withTestCreateWireWorkspaceBody(body))),
				WithSampleObservationReqFilter(&pb.Sample_Filter{
					Sample: &pb.Ref_Sample{
						Id: &pb.Sample_ID{
							Name: "foo",
						},
					},
					Subsets: []*pb.Sample_Subset_ID{
						{Name: "foo"},
					},
					Seed: 1234,
				}),
			},
			expected: &SampleObservationReq{
				Ws:   testCreateWireWorkspace(t, withTestCreateWireWorkspaceBody(body)),
				Func: SampleFuncAll,
				Filter: &pb.Sample_Filter{
					Sample: &pb.Ref_Sample{
						Id: &pb.Sample_ID{
							Name: "foo",
						},
					},
					Subsets: []*pb.Sample_Subset_ID{
						{Name: "foo"},
					},
					Seed: 1234,
				},
			},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			req, err := NewSampleObservationReq(test.opts...)
			if test.shouldFail {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected.Ws.GetFlightplan(), req.Ws.GetFlightplan())
			require.Equal(t, test.expected.Filter, req.Filter)
		})
	}
}
