package status

import (
	"fmt"

	"google.golang.org/grpc/codes"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// Format returns the format status
func Format(cfg *pb.FormatRequest_Config, res *pb.FormatResponse) error {
	checkFailed := false
	var err error

	for _, out := range res.GetResponses() {
		if cfg.GetCheck() && out.GetChanged() {
			checkFailed = true
		}

		if !HasErrorDiags(out) {
			continue
		}

		err = Error(fmt.Sprintf("formatting %s failed", out.GetPath()), codes.Internal, err)
	}

	if HasErrorDiags(res) {
		err = Error("unable to format configuration", codes.Internal, err)
	}

	if checkFailed {
		return &ErrExit{
			ExitCode: 3,
			Err:      err,
			Code:     codes.Internal,
			Msg:      "check failed",
		}
	}

	return err
}
