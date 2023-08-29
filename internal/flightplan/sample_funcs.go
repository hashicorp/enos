package flightplan

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"math/rand"
	"slices"

	"github.com/hashicorp/enos/internal/random"
)

// SampleFuncAll takes a sample frame and returns all of the subsets elements as the observation.
// If the frame filter is not compatible with returning all an error will be returned.
func SampleFuncAll(ctx context.Context, frame *SampleFrame, r *rand.Rand) (*SampleObservation, error) {
	if frame == nil {
		return nil, fmt.Errorf("no sample frame was provided")
	}

	// Make sure that our sample frame adheres to all filter requirements.
	returnAll, err := frame.FilterValidate()
	if err != nil {
		return nil, err
	}

	if !returnAll {
		return nil, fmt.Errorf("filter is incompatible with returning all")
	}

	res := &SampleObservation{
		SampleFrame:        frame,
		SubsetObservations: map[string]*SampleSubsetObservation{},
	}

	for name, subFrame := range frame.SubsetFrames {
		res.SubsetObservations[name] = &SampleSubsetObservation{
			SampleSubsetFrame: subFrame,
			Matrix:            subFrame.Matrix,
		}
	}

	return res, nil
}

// SampleFuncPurposiveStratified takes a sample frame and random number generator and returns a new
// sample observation. We're purposive (or judgemental) in that our algorithm will prefer that each
// subset is represented before doing a stratified distribution by proportion. If there are any
// remainders after our purposive and stratified distributions then we'll distribute remaining
// elements across subsets evenly by order of subset remaining capacity.
func SampleFuncPurposiveStratified(ctx context.Context, frame *SampleFrame, r *rand.Rand) (*SampleObservation, error) {
	if frame == nil {
		return nil, fmt.Errorf("no sample frame was provided")
	}

	if r == nil {
		return nil, fmt.Errorf("no source of random entropy was provided")
	}

	// Make sure that our sample frame adheres to all filter requirements.
	returnAll, err := frame.FilterValidate()
	if err != nil {
		return nil, err
	}

	// Our filter maximum is larger or equal to the entire sample field so we can simply return the whole
	// thing.
	if returnAll {
		return SampleFuncAll(ctx, frame, r)
	}

	// Determine our sample boundaries.
	min, max, err := frame.FilterMinMax()
	if err != nil {
		return nil, err
	}

	// Create our sample specifications for each subset. We'll do this by converting our frame subsets
	// into specifications and then allocating elements using the our purposive stratfied algorithm.
	subsetSpecs := sampleFrameToSubsetSpecs(frame)
	err = sampleAllocatePurposiveStratified(subsetSpecs, max, r)
	if err != nil {
		return nil, err
	}

	res := &SampleObservation{
		SampleFrame: frame,
	}

	// Take an observation of our frame using our specs as a guide. Do a simple random sample from
	// each subset.
	res.SubsetObservations, err = sampleObserveSimpleRandom(frame, subsetSpecs, r)
	if err != nil {
		return nil, err
	}

	if res.Size() < min {
		return nil, fmt.Errorf("sample observation size of %d does not satisfy minimum requirement: %d",
			res.Size(), min,
		)
	}

	return res, nil
}

// sampleSubsetObsSpec is a specification that describes how many elements to take for given subset.
type sampleSubsetObsSpec struct {
	name  string // the subset we represent
	space int32  // the size of the subset
	taken int32  // how many we should take
}

// take increases the specs representation of how many elements the subset should take while also
// decreasing our remaining space. If we've run out of space, i.e. over repesented the subset,
// return an error.
func (s *sampleSubsetObsSpec) take(i int32) error {
	if s.space < i {
		return fmt.Errorf(
			"cannot allocate %d in subset %s, space remaining: %d",
			i, s.name, s.space,
		)
	}

	s.taken += i
	s.space -= i

	return nil
}

// size is the total size of free space and taken elements.
func (s *sampleSubsetObsSpec) size() int32 {
	return s.space + s.taken
}

// Convert our sample frame into a collection of subset specs that we can use for determine
// how many elements we should take from each subset.
func sampleFrameToSubsetSpecs(frame *SampleFrame) []*sampleSubsetObsSpec {
	subsetSpecs := []*sampleSubsetObsSpec{}
	for name := range frame.SubsetFrames {
		subFrame := frame.SubsetFrames[name]
		subsetSpecs = append(subsetSpecs, &sampleSubsetObsSpec{
			name:  name,
			space: subFrame.Size(),
		})
	}
	sortSubsetSpecsByRemainingCapSpace(subsetSpecs)

	return subsetSpecs
}

// Covert our intermediate representation into a SampleSubsetObservations.
func sampleObserveSimpleRandom(
	frame *SampleFrame,
	subsetSpecs []*sampleSubsetObsSpec,
	r *rand.Rand,
) (
	SampleSubsetObservations,
	error,
) {
	res := SampleSubsetObservations{}
	if len(subsetSpecs) < 1 {
		return res, nil
	}

	if frame == nil {
		return nil, fmt.Errorf("a frame is required to observe subset samples")
	}

	if r == nil {
		return nil, fmt.Errorf("a random number source is required to observe subset samples")
	}

	sortSubsetSpecsByCapTaken(subsetSpecs)
	for i := range subsetSpecs {
		if subsetSpecs[i].taken == 0 {
			continue
		}

		subset, ok := frame.SubsetFrames[subsetSpecs[i].name]
		if !ok {
			return nil, fmt.Errorf("expected to sample from frame %s but it was not found in frame", subsetSpecs[i].name)
		}
		obs, err := subset.ObserveSimpleRandom(subsetSpecs[i].taken, r)
		if err != nil {
			return nil, err
		}
		res[subsetSpecs[i].name] = obs
	}

	return res, nil
}

func sortSubsetSpecsByCapTaken(subsetSpecs []*sampleSubsetObsSpec) {
	slices.SortStableFunc(subsetSpecs, func(a, b *sampleSubsetObsSpec) int {
		// Reverse sort so we put the largest at the top of the slice.
		if n := cmp.Compare(b.taken, a.taken); n != 0 {
			return n
		}

		return cmp.Compare(a.name, b.name)
	})
}

func sortSubsetSpecsByRemainingCapSpace(subsetSpecs []*sampleSubsetObsSpec) {
	slices.SortStableFunc(subsetSpecs, func(a, b *sampleSubsetObsSpec) int {
		// Reverse sort so we put the largest at the top of the slice.
		if n := cmp.Compare(b.space, a.space); n != 0 {
			return n
		}

		return cmp.Compare(a.name, b.name)
	})
}

func sampleAllocatePurposiveStratified(subsetSpecs []*sampleSubsetObsSpec, take int32, r *rand.Rand) error {
	if len(subsetSpecs) == 0 {
		return nil
	}

	sortSubsetSpecsByRemainingCapSpace(subsetSpecs)
	takePurposive := int32(len(subsetSpecs))
	if take < takePurposive {
		takePurposive = take
	}

	// Try and represent each subset in the sample allocating a purposive sample across the subsets.
	err := sampleAllocatePurposiveAllSubsetRepresented(subsetSpecs, takePurposive, r)
	if err != nil {
		return err
	}

	if take == takePurposive {
		// We already took all we need from our purposive sample.
		return nil
	}

	// Determine how many more we need to allocate.
	remain := take - takePurposive

	// Allocate our remaining using the stratified algorithm.
	remain, err = sampleAllocateStratified(subsetSpecs, remain)
	if err != nil {
		return err
	}

	if remain < 1 {
		return nil
	}

	// Allocate any remaining across all samples by order of remaining cap space.
	return sampleAllocatePurposiveCapSpace(subsetSpecs, remain)
}

// sampleAllocateStratified takes our subsetSpecs, how many we should attempt to allocate, and a random
// number source. It will then allocate in a stratified manner according to subset size relative
// to that of the entire frame size. Due to rounding any allocations that we were not able to make
// will be returned. If the take is invalid an error will be returned.
func sampleAllocateStratified(subsetSpecs []*sampleSubsetObsSpec, take int32) (int32, error) {
	if len(subsetSpecs) < 1 || take < 1 {
		return 0, nil
	}

	var frameSize int32
	var frameRemainingSpace int32
	for i := range subsetSpecs {
		frameSize += subsetSpecs[i].size()
		frameRemainingSpace += subsetSpecs[i].space
	}

	if take > frameRemainingSpace {
		return 0, fmt.Errorf(
			"cannot take %d from %d subsets with %d space remaining to allocate",
			take, len(subsetSpecs), frameRemainingSpace,
		)
	}

	// Iterate over out subsetSpecs and take a proportional amount based on the subset size relative
	// to total frame size. We'll order by cap space to ensure we'll take from the largest subsets
	// before smaller ones.
	sortSubsetSpecsByRemainingCapSpace(subsetSpecs)
	took := int32(0)
	for i := range subsetSpecs {
		// Calculate how many we should take for the subset frame
		subTake := int32(math.Round(float64(take) * (float64(subsetSpecs[i].size()) / float64(frameSize))))
		if subTake < 1 {
			continue
		}

		// Make sure we don't over-represent the subset
		canTake := subsetSpecs[i].space - subTake
		if subTake > canTake {
			subTake = canTake
		}

		// Make sure we don't try and take more than remains
		if take < (took + subTake) {
			subTake = take - took
		}

		// Make sure that we still have some to take
		if subTake < 1 {
			continue
		}

		err := subsetSpecs[i].take(subTake)
		if err != nil {
			return 0, err
		}

		took += subTake
		if (take - took) < 1 {
			// We're done.
			return 0, nil
		}
	}

	take -= took

	return take, nil
}

// sampleAllocatePurposiveAllSubsetRepresented samples across the subsets by taking one for each to
// ensure all are represented. If the take limit is less than our subsets we'll randomly select which
// subsets to take. If the take limit exceeds the subsets an error will be returned.
func sampleAllocatePurposiveAllSubsetRepresented(subsetSpecs []*sampleSubsetObsSpec, take int32, r *rand.Rand) error {
	if len(subsetSpecs) < 1 {
		return nil
	}

	if r == nil {
		return fmt.Errorf("a random number source is required to observe subset samples")
	}

	// Get the indices we wish to take from. If our take is less the frame width we'll select randomly
	// from our subsets to that limit. If the take is more an error will we returned as we're only
	// trying to make it representative.
	indices, err := random.SampleInt(int(take), len(subsetSpecs), r)
	if err != nil {
		return err
	}

	for i := range indices {
		err := subsetSpecs[indices[i]].take(1)
		if err != nil {
			return err
		}
	}

	return nil
}

// sampleAllocatePurposiveCapSpace distributes the take across subsets by order of remaining cap
// space. This is useful for allocating smaller remainders to the largest subsets.
func sampleAllocatePurposiveCapSpace(subsetSpecs []*sampleSubsetObsSpec, take int32) error {
	if take < 1 {
		return nil
	}

	for {
		if take == 0 {
			return nil
		}

		sortSubsetSpecsByRemainingCapSpace(subsetSpecs)
		for i := range subsetSpecs {
			if subsetSpecs[i].space == 0 {
				// We don't have any more space. This should never happen but we'll check for it anyway.
				return fmt.Errorf("unable to allocate subset elements")
			}

			if err := subsetSpecs[i].take(1); err == nil {
				take--
			}

			if take == 0 {
				return nil
			}
		}
	}
}
