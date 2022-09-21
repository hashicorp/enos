package status

import (
	"fmt"
	"strings"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// Error takes a message and optional errors to wrap and returns a new error
func Error(msg string, errs ...error) error {
	err := fmt.Errorf(msg)
	for _, err2 := range errs {
		if err2 != nil {
			err = fmt.Errorf("%s: %w", err.Error(), err2)
		}
	}

	return err
}

// ErrExit is an error that contains requested special exit behavior
type ErrExit struct {
	Err      error
	ExitCode int
	Msg      string
}

func (e *ErrExit) Unwrap() error {
	return e.Err
}

func (e *ErrExit) Error() string {
	return Error(e.Msg, e.Err).Error()
}

// ErrDiagnostic is an error that can carry diagnostics information
type ErrDiagnostic struct {
	Diags          []*pb.Diagnostic
	DiagStringOpts []diagnostics.StringOpt
	Err            error
}

// Error returns a joined message from all diagnostics errors
func (e *ErrDiagnostic) Error() string {
	if e.Diags == nil {
		if e.Err != nil {
			return e.Err.Error()
		}
		return ""
	}

	msg := strings.Builder{}
	for _, diag := range e.Diags {
		_, _ = msg.WriteString(diagnostics.String(diag, e.DiagStringOpts...))
	}

	return msg.String()
}

// Unwrap returns the wrapped error
func (e *ErrDiagnostic) Unwrap() error {
	return e.Err
}
