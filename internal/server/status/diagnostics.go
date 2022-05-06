package status

import (
	"fmt"

	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

type resWithDiags interface {
	GetDiagnostics() []*pb.Diagnostic
}

// HasErrorDiags returns whether or not the response has error diagnostics
func HasErrorDiags(res resWithDiags) bool {
	if res == nil {
		return false
	}

	return diagnostics.HasErrors(res.GetDiagnostics())
}

// Error takes a message, gRPC error code, and an optional error to wrap
// and returns a new error
func Error(msg string, code codes.Code, errs ...error) error {
	for _, err := range errs {
		if err != nil {
			msg = fmt.Sprintf("%s: %s", msg, err.Error())
		}
	}

	return grpcstatus.Errorf(code, msg)
}
