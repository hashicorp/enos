package flightplan

import (
	"fmt"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
)

// ErrDiagnostic is an error that can carry diagnostics information
type ErrDiagnostic struct {
	Diags hcl.Diagnostics
	Files map[string]*hcl.File
	Err   error
}

// Error returns a joined message from all diagnostics errors
func (e *ErrDiagnostic) Error() string {
	msg := strings.Builder{}
	if e.Diags == nil {
		return e.Err.Error()
	}

	for i, err := range e.Diags.Errs() {
		if i == 0 {
			msg.WriteString(err.Error())
		} else {
			msg.WriteString(fmt.Sprintf(": %s", err.Error()))
		}
	}

	return msg.String()
}

// Unwrap returns the wrapped error
func (e *ErrDiagnostic) Unwrap() error {
	return e.Err
}
