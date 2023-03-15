package status

import (
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ListSamples returns the status response for a sample list.
func ListSamples(failOnWarn bool, res *pb.ListSamplesResponse) error {
	if HasFailed(failOnWarn, res, res.GetDecode()) {
		return Error("failed to list samples")
	}

	return nil
}

// ShowSampleObservation returns status of the sample observation.
func ShowSampleObservation(failOnWarn bool, res *pb.ObserveSampleResponse) error {
	if HasFailed(failOnWarn, res, res.GetDecode()) {
		return Error("failed to show sample observation")
	}

	return nil
}
