package acceptance

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func TestAcc_Cmd_Fmt(t *testing.T) {
	t.Parallel()

	enos := newAcceptanceRunner(t)

	path, err := filepath.Abs("./invalid_scenarios/scenario_not_formatted")
	require.NoError(t, err)

	cmd := fmt.Sprintf("fmt %s -d -c --format json", path)
	out, err := enos.run(context.Background(), cmd)
	target := &exec.ExitError{}
	require.Error(t, err)
	if errors.As(err, &target) {
		require.Equal(t, 3, target.ProcessState.ExitCode())
	} else {
		t.Fatal("fmt did not return exit code 3 on changed")
	}

	expected := &pb.FormatResponse{
		Responses: []*pb.FormatResponse_Response{
			{
				Path:    filepath.Join(path, "enos.hcl"),
				Changed: true,
			},
			{
				Path: filepath.Join(path, "enos.vars.hcl"),
			},
		},
	}

	got := &pb.FormatResponse{}
	require.NoErrorf(t, protojson.Unmarshal(out, got), string(out))
	require.Len(t, got.GetResponses(), len(expected.GetResponses()))
	for i := range expected.GetResponses() {
		got := got.GetResponses()[i]
		expected := expected.GetResponses()[i]
		require.Equal(t, expected.GetPath(), got.GetPath())
		require.Equal(t, expected.GetChanged(), got.GetChanged())
	}
}
