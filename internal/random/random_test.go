// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package random

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_SampleInt(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		size       int
		rng        int
		expected   []int
		shouldFail bool
	}{
		"size bigger than range": {
			shouldFail: true,
			size:       10,
			rng:        9,
		},
		"size is exactly range": {
			size:     10,
			rng:      10,
			expected: []int{0, 7, 3, 5, 9, 6, 2, 1, 8, 4},
		},
		"size is zero": {
			size:     0,
			rng:      10,
			expected: []int{},
		},
		"size is negative": {
			shouldFail: true,
			size:       -1,
			rng:        10,
		},
		"size is less than range": {
			size:     8,
			rng:      10,
			expected: []int{8, 5, 0, 2, 6, 1, 9, 4},
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			//nolint:gosec// G404 we're using a weak random number generator because secure random
			// numbers are not needed for this use case.
			r := rand.New(rand.NewSource(3456))
			res, err := SampleInt(test.size, test.rng, r)
			if test.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, res)
			}
		})
	}
}
