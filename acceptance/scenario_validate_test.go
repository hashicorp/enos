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

func TestAcc_Cmd_Scenario_Validate(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		dir  string
		out  *pb.ValidateScenariosConfigurationResponse
		fail bool
	}{
		{
			dir: "scenarios/scenario_list_pass_3",
			out: &pb.ValidateScenariosConfigurationResponse{},
		},
		{
			dir:  "invalid_scenarios/scenario_list_fail_malformed",
			fail: true,
		},
		{
			dir:  "invalid_scenarios/sample_empty_frame",
			fail: true,
		},
	} {
		t.Run(test.dir, func(t *testing.T) {
			t.Parallel()
			enos := newAcceptanceRunner(t)
			path, err := filepath.Abs(filepath.Join("./", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario validate --chdir %s --format json", path)
			fmt.Println(path)
			out, _, err := enos.run(context.Background(), cmd)
			if test.fail {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			got := &pb.ValidateScenariosConfigurationResponse{}
			require.NoError(t, protojson.Unmarshal(out, got))
		})
	}
}

func TestAcc_Cmd_Scenario_Validate_filtered(t *testing.T) {
	t.Parallel()

	for filter, test := range map[string]struct {
		out  *pb.ValidateScenariosConfigurationResponse
		fail bool
	}{
		"test backend:raft": {
			out: &pb.ValidateScenariosConfigurationResponse{},
		},
		"test backend:not_a_variant_value": {
			fail: true,
		},
		"test not_a_variant:raft": {
			fail: true,
		},
	} {
		t.Run(filter, func(t *testing.T) {
			t.Parallel()
			enos := newAcceptanceRunner(t)
			path, err := filepath.Abs(filepath.Join("./", "scenarios/scenario_list_pass_3"))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario validate %s --chdir %s --format json", filter, path)
			fmt.Println(path)
			out, _, err := enos.run(context.Background(), cmd)
			if test.fail {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			got := &pb.ValidateScenariosConfigurationResponse{}
			require.NoError(t, protojson.Unmarshal(out, got))
		})
	}
}
