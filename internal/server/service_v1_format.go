package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Format does formatting on Enos configuration
func (s *ServiceV1) Format(
	ctx context.Context,
	req *pb.FormatRequest,
) (
	*pb.FormatResponse,
	error,
) {
	res := &pb.FormatResponse{
		Responses: []*pb.FormatResponse_Response{},
	}

	for path, b := range req.GetFiles() {
		r := &pb.FormatResponse_Response{
			Path: path,
		}
		// Make sure we can parse it as valid HCL, otherwise whatever we'd
		// format would likely render it even more broken.
		_, diags := hclwrite.ParseConfig(b, path, hcl.InitialPos)
		if diags.HasErrors() {
			r.Diagnostics = diagnostics.FromHCL(nil, diags)
			res.Responses = append(res.Responses, r)
			continue
		}

		formatted := hclwrite.Format(b)
		if bytes.Equal(b, formatted) {
			// If nothing has changed we can move on
			res.Responses = append(res.Responses, r)
			continue
		}

		r.Changed = true

		if req.GetConfig().GetDiff() {
			edits := myers.ComputeEdits(
				span.URIFromPath(path),
				string(b),
				string(formatted),
			)
			r.Diff = strings.TrimSuffix(
				fmt.Sprint(gotextdiff.ToUnified("old", "new", string(b), edits)),
				"\n",
			)
		}

		if path == "STDIN" && req.GetConfig().GetWrite() {
			r.Body = strings.TrimSuffix(string(formatted), "\n")
		}

		if path != "STDIN" && req.GetConfig().GetWrite() && !req.GetConfig().GetCheck() {
			f, err := os.OpenFile(path, os.O_RDWR, 0o755)
			if err != nil {
				res.Diagnostics = diagnostics.FromErr(err)
				res.Responses = append(res.Responses, r)
				continue
			}
			defer f.Close()

			_, err = io.Copy(f, bytes.NewReader(formatted))
			if err != nil {
				res.Diagnostics = diagnostics.FromErr(err)
				res.Responses = append(res.Responses, r)
				continue
			}
		}

		res.Responses = append(res.Responses, r)
	}

	return res, nil
}
