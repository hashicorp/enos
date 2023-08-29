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

func TestAcc_Cmd_Scenario_Validate(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		dir  string
		out  *pb.ValidateScenariosConfigurationResponse
		fail bool
	}{
		{
			dir: "scenarios/scenario_list_pass_0",
			out: &pb.ValidateScenariosConfigurationResponse{},
		},
		{
			dir:  "invalid_scenarios/scenario_list_fail_malformed",
			fail: true,
		},
	} {
		test := test
		t.Run(test.dir, func(t *testing.T) {
			t.Parallel()
			enos := newAcceptanceRunner(t)
			path, err := filepath.Abs(filepath.Join("./", test.dir))
			require.NoError(t, err)
			cmd := fmt.Sprintf("scenario validate --chdir %s --format json", path)
			fmt.Println(path)
			out, err := enos.run(context.Background(), cmd)
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
