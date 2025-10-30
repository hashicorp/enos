// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package status

import (
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
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
