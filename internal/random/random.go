package random

import (
	"fmt"
	"math/rand"
)

// SampleInt takes a sample size, the maximum integer for a half-opened interval [0,n) and a random
// number source and returns a slize of sampled intergers within the closed interval. The sample
// integers are guaranteed to be unique if a sample is possible. If the range sample size is larger
// that the interval range an error will be returned.
func SampleInt(take, rng int, r *rand.Rand) ([]int, error) {
	if take > rng {
		return nil, fmt.Errorf("sample size %d cannot exceed sample range %d", take, rng)
	}

	if take < 0 {
		return nil, fmt.Errorf("sample size %d cannot be a negative number", take)
	}

	if take == 0 {
		return []int{}, nil
	}

	// Handle a sample where all elements have to be used but we want them shuffled.
	if take == rng {
		res := make([]int, take)
		for i := range res {
			res[i] = i
		}

		r.Shuffle(take, func(i, j int) {
			res[i], res[j] = res[j], res[i]
		})

		return res, nil
	}

	// We'll use a list based option rather than trying a set.
	possible := make([]int, rng)
	for i := range possible {
		possible[i] = i
	}

	cardinality := 0
	res := make([]int, take)
	for {
		i := r.Intn(len(possible))
		res[cardinality] = possible[i]
		possible = append(possible[:i], possible[i+1:]...)
		cardinality++

		if cardinality == take {
			break
		}
	}

	return res, nil
}
