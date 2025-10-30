// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStat(t *testing.T) {
	t.Parallel()

	out, err := Stat(context.Background(), WithGC())
	require.NoError(t, err)
	require.NotEmpty(t, out.String())
	require.NotZero(t, out.Free())
	require.NotZero(t, out.Used())
	require.NotZero(t, out.Total())
}
