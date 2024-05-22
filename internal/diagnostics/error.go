// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package diagnostics

import (
	"strings"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

var _ error = (*Error)(nil)

func NewError() *Error {
	return &Error{}
}

// Error is an error that can carry diagnostics information.
type Error struct {
	Diags          []*pb.Diagnostic
	DiagStringOpts []StringOpt
	Err            error
}

// Error returns a joined message from all diagnostics errors.
func (e *Error) Error() string {
	if e.Diags == nil {
		if e.Err != nil {
			return e.Err.Error()
		}

		return ""
	}

	msg := strings.Builder{}

	if e.Err != nil {
		msg.WriteString(e.Err.Error())
	}

	for _, diag := range e.Diags {
		_, _ = msg.WriteString(String(diag, e.DiagStringOpts...))
	}

	return msg.String()
}

// Unwrap returns the wrapped error.
func (e *Error) Unwrap() error {
	return e.Err
}

// SetStringOpts allows configuring the stringer opts on the error. This allows the caller
// to determine the formatting of the error message if diagnostics are preset.
func (e *Error) SetStringOpts(opts ...StringOpt) {
	e.DiagStringOpts = opts
}
