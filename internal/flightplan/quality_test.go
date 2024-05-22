// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// Test_Decode_Quality tests module decoding.
func Test_Decode_Quality(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		desc     string
		hcl      string
		expected *FlightPlan
		fail     bool
	}{
		{
			desc: "valid",
			hcl: `
module "backend" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.11.0"
}

quality "passes_tests" {
	description = "The software passes the tests"
}

quality "state_is_durable" {
	description = "The state is durable"
}

scenario "basic" {
  step "first" {
    module = module.backend
  }
}
`,
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				Modules: []*Module{
					{
						Name:    "backend",
						Source:  "terraform-aws-modules/vpc/aws",
						Version: "3.11.0",
					},
				},
				Qualities: []*Quality{
					{
						Name:        "passes_tests",
						Description: "The software passes the tests",
					},
					{
						Name:        "state_is_durable",
						Description: "The state is durable",
					},
				},
				ScenarioBlocks: DecodedScenarioBlocks{
					{
						Name: "basic",
						Scenarios: []*Scenario{
							{
								Name:         "basic",
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name: "first",
										Module: &Module{
											Name:    "backend",
											Source:  "terraform-aws-modules/vpc/aws",
											Version: "3.11.0",
											Attrs:   map[string]cty.Value{},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "invalid identifier",
			fail: true,
			hcl: `
module "backend" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.11.0"
}

quality "passes tests" {
	description = "The software passes the tests"
}

scenario "basic" {
  step "first" {
    module = module.backend
  }
}
`,
		},
		{
			desc: "no description",
			fail: true,
			hcl: `
module "backend" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.11.0"
}

quality "passes_tests" {
}

scenario "basic" {
  step "first" {
    module = module.backend
  }
}
`,
		},
		{
			desc: "defined more than once",
			fail: true,
			hcl: `
module "backend" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.11.0"
}

quality "passes_tests" {
	description = "The software passes the tests"
}

quality "passes_tests" {
	description = "The software passes the tests"
}

scenario "basic" {
  step "first" {
    module = module.backend
  }
}
`,
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			fp, err := testDecodeHCL(t, []byte(test.hcl), DecodeTargetAll)
			if test.fail {
				require.Error(t, err)

				return
			}
			require.NoError(t, err)
			testRequireEqualFP(t, fp, test.expected)
		})
	}
}

// TestCompareQualityProto tests that our comparison function is able to stable sort.
func TestCompareQualityProto(t *testing.T) {
	t.Parallel()

	expected := []*pb.Quality{
		{Name: "aaaa", Description: "aaaa"},
		{Name: "aaaa", Description: "bbbb"},
		{Name: "bbbb", Description: "aaaa"},
		{Name: "bbbb", Description: "bbbb"},
	}
	for name, qualities := range map[string][]*pb.Quality{
		"presorted": {
			&pb.Quality{Name: "aaaa", Description: "aaaa"},
			&pb.Quality{Name: "aaaa", Description: "bbbb"},
			&pb.Quality{Name: "bbbb", Description: "aaaa"},
			&pb.Quality{Name: "bbbb", Description: "bbbb"},
		},
		"reversed": {
			&pb.Quality{Name: "bbbb", Description: "bbbb"},
			&pb.Quality{Name: "bbbb", Description: "aaaa"},
			&pb.Quality{Name: "aaaa", Description: "bbbb"},
			&pb.Quality{Name: "aaaa", Description: "aaaa"},
		},
		"mixed": {
			&pb.Quality{Name: "bbbb", Description: "bbbb"},
			&pb.Quality{Name: "aaaa", Description: "aaaa"},
			&pb.Quality{Name: "bbbb", Description: "aaaa"},
			&pb.Quality{Name: "aaaa", Description: "bbbb"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			slices.SortStableFunc(qualities, CompareQualityProto)
			for i := range expected {
				require.EqualValues(t, expected[i].GetName(), qualities[i].GetName())
				require.EqualValues(t, expected[i].GetDescription(), qualities[i].GetDescription())
			}
		})
	}
}
