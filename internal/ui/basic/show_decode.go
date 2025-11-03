// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"github.com/hashicorp/enos/internal/ui/status"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// ShowDecode shows the human friendly view of decoding.
func (v *View) ShowDecode(res *pb.DecodeResponse, incremental bool) error {
	if res == nil {
		return nil
	}

	v.writeDecodeResponse(res)

	return status.Decode(v.Settings().GetFailOnWarnings(), res)
}
