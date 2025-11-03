// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package status

import pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"

// ScenariosValidateConfig returns the status response for a flight plan validation.
func ScenariosValidateConfig(failOnWarn bool, res *pb.ValidateScenariosConfigurationResponse) error {
	if HasFailed(failOnWarn, res, res.GetDecode(), res.GetSampleDecode()) {
		return Error("scenario configuration is not valid")
	}

	return nil
}
