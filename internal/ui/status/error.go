// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package status

import (
	"errors"
	"fmt"
)

// Error takes a message and optional errors to wrap and returns a new error.
func Error(msg string, errs ...error) error {
	err := errors.New(msg)
	for _, err2 := range errs {
		if err2 != nil {
			err = fmt.Errorf("%s: %w", err.Error(), err2)
		}
	}

	return err
}

// ErrExit is an error that contains requested special exit behavior.
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
