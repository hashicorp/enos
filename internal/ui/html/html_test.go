// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package html

import (
	"testing"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"

	"github.com/stretchr/testify/require"
)

// TestShowScenarioOutline tests showing the scenario outline.
func TestShowScenarioOutline(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		v func() *View
	}{
		"raw does not panic": {
			func() *View { v := &View{}; return v },
		},
		"contructor does not panic": {
			func() *View { v, err := New(); require.NoError(t, err); return v },
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, test.v().ShowScenarioOutline(&pb.OutlineScenariosResponse{}))
		})
	}
}
