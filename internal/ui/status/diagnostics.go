// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package status

import (
	"github.com/hashicorp/enos/internal/diagnostics"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// ResWithDiags is an interface of a response that has diagnostics.
type ResWithDiags interface {
	GetDiagnostics() []*pb.Diagnostic
}

// HasErrorDiags returns whether or not the response has error diagnostics.
func HasErrorDiags(res ...ResWithDiags) bool {
	if len(res) < 1 {
		return false
	}

	return diagnostics.HasErrors(combinedResWithDiags(res))
}

// HasWarningDiags returns whether or not the response has warning diagnostics.
func HasWarningDiags(res ...ResWithDiags) bool {
	if res == nil {
		return false
	}

	return diagnostics.HasWarnings(combinedResWithDiags(res))
}

// HasFailed takes a boolean which determines whether or not the diagnostics
// failed or not.
func HasFailed(failOnWarn bool, res ...ResWithDiags) bool {
	if HasErrorDiags(res...) {
		return true
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
