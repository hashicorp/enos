// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
)

type errNotDefinedInCtx struct {
	Name string
	Err  error
}

func (e *errNotDefinedInCtx) Error() string {
	return fmt.Sprintf("an eval context variable with name %s has not been defined", e.Name)
}

func (e *errNotDefinedInCtx) Unwrap() error {
	return e.Err
}

func findEvalContextVariable(name string, baseCtx *hcl.EvalContext) (cty.Value, error) {
	var val cty.Value

	// Search through the eval context chain until we find a variable that
	// matches our name
	for ctx := baseCtx; ctx != nil; ctx = ctx.Parent() {
		if ctx == nil {
			// We've run out of eval contexts to search so we'll break out and
			// return an error
			break
		}

		var ok bool
		val, ok = ctx.Variables[name]
		if ok {
			return val, nil
		}
	}

	return val, &errNotDefinedInCtx{Name: name}
}
