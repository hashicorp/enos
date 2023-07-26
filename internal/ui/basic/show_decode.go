package basic

import (
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowDecode shows the human friendly view of decoding.
func (v *View) ShowDecode(res *pb.DecodeResponse, incremental bool) error {
	if res == nil {
		return nil
	}

	v.writeDecodeResponse(res)

	return status.Decode(v.Settings().GetFailOnWarnings(), res)
}
