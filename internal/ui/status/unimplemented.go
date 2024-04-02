// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package status

import "fmt"

// Unimplemented returns an unimplemented error.
func Unimplemented(msg string) error {
	return fmt.Errorf("not implemented: %s", msg)
}
