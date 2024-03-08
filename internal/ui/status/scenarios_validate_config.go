// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package status

import "github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"

// ScenariosValidateConfig returns the status response for a flight plan validation.
func ScenariosValidateConfig(failOnWarn bool, res *pb.ValidateScenariosConfigurationResponse) error {
	if HasFailed(failOnWarn, res, res.GetDecode()) {
		return Error("scenario configuration is not valid")
	}

	return nil
}
