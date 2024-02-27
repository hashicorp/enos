// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"slices"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/hashicorp/enos/internal/random"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// A sample field is a collection of subset fields one-or-more subsets.
type SampleFrame struct {
	*Sample
	Filter       *pb.Sample_Filter
	SubsetFrames SampleSubsetFrames
}

func (s *SampleFrame) Ref() *pb.Ref_Sample {
	if s == nil || s.Sample == nil {
		return nil
	}

	return &pb.Ref_Sample{
		Id: &pb.Sample_ID{
			Name: s.Sample.Name,
		},
	}
}

// Elements takes a SubsetFrames name, a random number source, and optionally a matrix whose
// vertices refer elements in the sample subset frame to include. If no matrix is given the
// entire subset frame will be returned.
func (s *SampleFrame) Elements(subsetFrameName string, r *rand.Rand, m *Matrix) ([]*pb.Sample_Element, error) {
	if s == nil || s.SubsetFrames == nil {
		return nil, errors.New("cannot expand elements without subset frames")
	}

	subsetFrame, ok := s.SubsetFrames[subsetFrameName]
	if !ok {
		return nil, fmt.Errorf("no subset frame with name %s", subsetFrameName)
	}

	var err error
	sampleVals := map[string]cty.Value{}
	subsetVals := map[string]cty.Value{}

	if s != nil && s.Sample != nil {
		sampleVals, err = sampleAttrVals(s.Sample.Attributes)
		if err != nil {
			return nil, err
		}
		if sampleVals == nil {
			return nil, errors.New("cannot select elements from sample frame with no values")
		}
	}

	if subsetFrame != nil && subsetFrame.SampleSubset != nil {
		subsetVals, err = sampleAttrVals(subsetFrame.SampleSubset.Attributes)
		if err != nil {
			return nil, err
		}
		if subsetVals == nil {
			return nil, errors.New("cannot select elements from sample subset frame with no values")
		}
	}

	if subsetFrame.Matrix == nil && m != nil {
		return nil, fmt.Errorf(
			"frame for subset %s has no matrix but requested elements for %s",
			subsetFrameName, m.String(),
		)
	}

	var subElements []*pb.Sample_Element
	if subsetFrame.Matrix == nil {
		subElements = sampleElementsFor(s, subsetFrame, nil)
	} else {
		matrix := subsetFrame.Matrix
		if m != nil {
			matrix = m.IntersectionContainsUnordered(matrix)
		}
		subElements = sampleElementsFor(s, subsetFrame, matrix.GetVectors()...)
	}

	// Merge the subset vals into the sample vals. This will overwrite any outer keys.
	for key, val := range subsetVals {
		sampleVals[key] = val
	}

	return subElements, expandElementAttrs(subElements, sampleVals, r)
}

// expandElementAttrs takes s list of sample elements, a map of attributes that can contain single
// or multiple values, and assigns the values to each element. In cases where attribute values are
// singular we will assign the value to each element. In vases where attributes have more than one
// value we'll randomly distribute the values across all elements. As with all sample functions
// that perform random actions we'll do it determinisitically.
func expandElementAttrs(elements []*pb.Sample_Element, vals map[string]cty.Value, r *rand.Rand) error {
	if vals == nil || len(vals) < 1 || len(elements) < 1 {
		return nil
	}

	// As we're dealing with a value map we'll sort the keys and iterate over them to make any
	// random expansion deterministic.
	elementValKeys := []string{}
	for k := range vals {
		elementValKeys = append(elementValKeys, k)
	}
	slices.Sort(elementValKeys)

	// Create a value bucket for each element and then distribute the attributes and values across
	// each element.
	elementVals := map[int]map[string]cty.Value{}
	for _, aKey := range elementValKeys {
		aVal := vals[aKey]

		// The value is singular so we'll add the value to each elements values.
		if !aVal.CanIterateElements() {
			for i := range elements {
				elmVal, ok := elementVals[i]
				if !ok {
					elmVal = map[string]cty.Value{}
				}
				elmVal[aKey] = aVal
				elementVals[i] = elmVal
			}

			continue
		}

		// The value has multiple values. We'll randomly distribute our attributes values across all
		// elements.
		if r == nil {
			return errors.New("no random number source given")
		}

		vals := aVal.AsValueSlice()
		if vals == nil {
			// This shouldn't happen but AsValueSlice can return nil
			continue
		}

		slices.SortStableFunc(vals, func(a, b cty.Value) int {
			return cmp.Compare(a.GoString(), b.GoString())
		})
		valIdx := make([]int, len(elements))
		maxIdx := len(vals) - 1
		nextIdxs, err := random.SampleInt(1, len(vals), r) // Randomly choose our first index
		if err != nil {
			return err
		}
		nextIdx := nextIdxs[0]

		// Distribute our possible evenly across element values
		for i := range valIdx {
			valIdx[i] = nextIdx
			if nextIdx == maxIdx {
				nextIdx = 0
			} else {
				nextIdx++
			}
		}

		// Now shuffle our indices
		r.Shuffle(len(valIdx), func(i, j int) {
			valIdx[i], valIdx[j] = valIdx[j], valIdx[i]
		})

		// Write our values
		for i, v := range valIdx {
			elmVal, ok := elementVals[i]
			if !ok {
				elmVal = map[string]cty.Value{}
			}
			elmVal[aKey] = vals[v]
			elementVals[i] = elmVal
		}
	}

	for i := range elementVals {
		var err error
		elements[i].Attributes, err = sampleAttrToProto(elementVals[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SampleFrame) FilterMin() (int32, error) {
	if s == nil {
		return 0, errors.New("get sample frame min: nil sample frame cannot have min")
	}

	if s.Filter == nil {
		return 0, errors.New("get sample frame min: sample does not have a filter")
	}

	return s.Filter.GetMinElements(), nil
}

func (s *SampleFrame) FilterMax() (int32, error) {
	if s == nil {
		return 0, errors.New("get sample frame max: nil sample frame cannot have max")
	}

	if s.Filter == nil {
		return 0, errors.New("get sample frame max: sample does not have a filter")
	}

	return s.Filter.GetMaxElements(), nil
}

func (s *SampleFrame) FilterPercentage() (float32, error) {
	if s == nil {
		return 0, errors.New("get sample frame pct: nil sample frame cannot have pct")
	}

	if s.Filter == nil {
		return 0, errors.New("get sample frame pct: sample does not have a filter")
	}

	return s.Filter.GetPercentage(), nil
}

func (s *SampleFrame) FilterMinMax() (int32, int32, error) {
	min, err := s.FilterMin()
	if err != nil {
		return 0, 0, err
	}

	max, err := s.FilterMax()
	if err != nil {
		return 0, 0, err
	}

	pct, err := s.FilterPercentage()
	if err != nil {
		return 0, 0, err
	}

	size := s.Size()
	if min > size {
		return 0, 0, fmt.Errorf("minimum requested frame size %d is less that total frame size %d", min, size)
	}

	// Handle a cases where we don't have a percentage or max set.
	if max < 0 && pct < 0 {
		return min, size, nil
	}

	// Get out actual maximum from our max setting
	if max < 0 || max > size {
		max = size
	}

	// Handle cases where percentage rule has been set.
	if pct > 0 {
		pctMax := math.Round(float64(pct/100) * float64(size))

		// We have configured both a max upper bound and a percentage. Go with whatever is smaller.
		max = int32(math.Min(float64(max), pctMax))
	}

	return min, max, nil
}

func (s *SampleFrame) Size() int32 {
	if s == nil {
		return 0
	}

	if s.SubsetFrames == nil || len(s.SubsetFrames) < 1 {
		return 0
	}

	size := int32(0)
	for _, v := range s.SubsetFrames {
		size += v.Size()
	}

	return size
}

func (s *SampleFrame) Keys() []string {
	if len(s.SubsetFrames) < 1 {
		return nil
	}

	keys := make([]string, len(s.SubsetFrames))
	i := 0
	for k := range s.SubsetFrames {
		keys[i] = k
		i++
	}

	slices.SortStableFunc(keys, func(a, b string) int {
		return cmp.Compare(a, b)
	})

	return keys
}

// FilterValidate compares the sample frame with our filter configuration settings. It returns a
// boolean if we should return the entire frame. It will raise an error if our frame is unable to
// meet out filter configuration.
func (s *SampleFrame) FilterValidate() (bool, error) {
	min, max, err := s.FilterMinMax()
	if err != nil {
		return false, err
	}

	size := s.Size()
	// Make sure our field is large enough to sample our minimum
	if max > size {
		return false, fmt.Errorf("sample frame size %d was less than sample filter minimum %d", size, min)
	}

	if max == size {
		return true, nil
	}

	return false, nil
}

func sampleAttrToProto(attrVals map[string]cty.Value) (*structpb.Struct, error) {
	if len(attrVals) == 0 {
		return structpb.NewStruct(nil)
	}

	vals := map[string]*structpb.Value{}
	for k, v := range attrVals {
		ctyEncoder := &ctyjson.SimpleJSONValue{Value: v}
		encoded, err := ctyEncoder.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("unable to marshal sample attribute wire value %s: %w",
				v.GoString(), err,
			)
		}
		val := &structpb.Value{}
		err = protojson.Unmarshal(encoded, val)
		if err != nil {
			return nil, err
		}

		vals[k] = val
	}

	return &structpb.Struct{Fields: vals}, nil
}

func sampleAttrVals(val cty.Value) (map[string]cty.Value, error) {
	if val.IsNull() {
		return map[string]cty.Value{}, nil
	}

	if !val.IsWhollyKnown() {
		return nil, errors.New("sample attribute values cannot be unknowable")
	}

	if !val.CanIterateElements() {
		return nil, fmt.Errorf("cannot iterate sample attributes type: %s. Must be a string keyed map of value", val.GoString())
	}

	return val.AsValueMap(), nil
}

func sampleElementsFor(
	sampleFrame *SampleFrame,
	subsetFrame *SampleSubsetFrame,
	vectors ...*Vector,
) []*pb.Sample_Element {
	// Handle no vectors
	if vectors == nil {
		elm := &pb.Sample_Element{}
		if sampleFrame != nil {
			elm.Sample = sampleFrame.Ref()
		}
		if subsetFrame != nil {
			elm.Subset = subsetFrame.Ref()

			if subsetFrame.ScenarioFilter != nil {
				scenario := NewScenario()
				scenario.Name = subsetFrame.ScenarioFilter.GetName()
				elm.Scenario = scenario.Ref()
			}
		}

		return []*pb.Sample_Element{elm}
	}

	// Handle multiple vectors
	res := []*pb.Sample_Element{}
	for i := range vectors {
		elm := &pb.Sample_Element{}
		if sampleFrame != nil {
			elm.Sample = sampleFrame.Ref()
		}
		if subsetFrame != nil {
			elm.Subset = subsetFrame.Ref()

			if subsetFrame.ScenarioFilter != nil {
				scenario := NewScenario()
				scenario.Name = subsetFrame.ScenarioFilter.GetName()
				scenario.Variants = vectors[i]
				elm.Scenario = scenario.Ref()
			}
		}
		res = append(res, elm)
	}

	return res
}
