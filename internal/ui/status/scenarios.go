package status

import (
	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// Decode returns the status for a fligth plan decode
func Decode(failOnWarn bool, res *pb.DecodeResponse) error {
	if HasFailed(failOnWarn, res) {
		return Error("failed to decode")
	}

	return nil
}

// OperationResponses returns the status multiple operations
func OperationResponses(failOnWarn bool, res *pb.OperationResponses) error {
	var err error
	for _, r := range res.GetResponses() {
		err = OperationResponse(failOnWarn, r)
		if err != nil {
			return err
		}
	}

	return nil
}

// OperationResponse returns the status for an operation
func OperationResponse(failOnWarn bool, res *pb.Operation_Response) error {
	if diagnostics.OpResFailed(failOnWarn, res) {
		// Return a status code here because the operation response UI should
		// handle drawing a summary for us, we only need to handle how we'll
		// exit.
		return &ErrExit{ExitCode: 1}
	}

	return nil
}

// ListScenarios returns the status response for a scenario list
func ListScenarios(failOnWarn bool, res *pb.ListScenariosResponse) error {
	if HasFailed(failOnWarn, res, res.GetDecode()) {
		return Error("failed to list scenarios")
	}

	return nil
}
