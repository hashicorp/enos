package status

import (
	"fmt"

	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ResWithDiags is an interface of a response that has diagnostics
type ResWithDiags interface {
	GetDiagnostics() []*pb.Diagnostic
}

// HasErrorDiags returns whether or not the response has error diagnostics
func HasErrorDiags(res ...ResWithDiags) bool {
	if len(res) < 1 {
		return false
	}

	return diagnostics.HasErrors(combinedResWithDiags(res))
}

// HasWarningDiags returns whether or not the response has warning diagnostics
func HasWarningDiags(res ...ResWithDiags) bool {
	if res == nil {
		return false
	}

	return diagnostics.HasWarnings(combinedResWithDiags(res))
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

// HasFailed takes a boolean which determines whether or not the diagnostics
// failed or not.
func HasFailed(failOnWarn bool, res ...ResWithDiags) bool {
	if HasErrorDiags(res...) {
		return false
	}

	return failOnWarn && HasWarningDiags(res...)
}

func combinedResWithDiags(res []ResWithDiags) []*pb.Diagnostic {
	diags := []*pb.Diagnostic{}
	if len(res) < 1 {
		return diags
	}

	for i := range res {
		diags = append(diags, res[i].GetDiagnostics()...)
	}

	return diags
}
