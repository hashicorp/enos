package flightplan

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ScenarioFilter is a filter for scenarios.
type ScenarioFilter struct {
	Name               string     // The scenario name
	Include            *Vector    // A scenanario filter broken include a matrix vector
	Exclude            []*Exclude // Explicit scenario/variant exclusions
	SelectAll          bool       // Get all scenarios and variants
	IntersectionMatrix *Matrix    // Like Include but can contain more than one Vector
}

// String returns the scenario filter as a string.
func (sf *ScenarioFilter) String() string {
	if sf == nil {
		return ""
	}

	str := sf.Name

	if sf.SelectAll {
		return str
	}

	for _, i := range sf.Include.Elements() {
		str = fmt.Sprintf("%s %s", str, i.String())
	}

	for _, e := range sf.Exclude {
		for _, elm := range e.Vector.Elements() {
			str = fmt.Sprintf("%s !%s:%s", str, elm.Key, elm.Val)
		}
	}

	return str
}

// ScenarioFilterOpt is a scenario filter constructor functional option.
type ScenarioFilterOpt func(*ScenarioFilter) error

// NewScenarioFilter takes in options and returns a new filter.
func NewScenarioFilter(opts ...ScenarioFilterOpt) (*ScenarioFilter, error) {
	f := &ScenarioFilter{}

	for _, opt := range opts {
		err := opt(f)
		if err != nil {
			return f, err
		}
	}

	return f, nil
}

// WithScenarioFilterName sets the filter name.
func WithScenarioFilterName(name string) ScenarioFilterOpt {
	return func(f *ScenarioFilter) error {
		f.Name = name

		return nil
	}
}

// WithScenarioFilterSelectAll makes the filter select all.
func WithScenarioFilterSelectAll() ScenarioFilterOpt {
	return func(f *ScenarioFilter) error {
		f.SelectAll = true

		return nil
	}
}

// WithScenarioFilterMatchingVariants makes the filter select only scenarios with
// variants that match the given values.
func WithScenarioFilterMatchingVariants(vec *Vector) ScenarioFilterOpt {
	return func(f *ScenarioFilter) error {
		f.Include = vec

		return nil
	}
}

// WithScenarioFilterParse parses the given filter.
func WithScenarioFilterParse(args []string) ScenarioFilterOpt {
	return func(f *ScenarioFilter) error {
		nf, err := ParseScenarioFilter(args)
		if err != nil {
			return err
		}

		f.Name = nf.Name
		f.Include = nf.Include
		f.Exclude = nf.Exclude
		f.SelectAll = nf.SelectAll

		return nil
	}
}

// WithScenarioFilterDecode decodes a filter from a proto Filter.
func WithScenarioFilterDecode(filter *pb.Scenario_Filter) ScenarioFilterOpt {
	return func(f *ScenarioFilter) error {
		f.FromProto(filter)

		return nil
	}
}

// WithScenarioFilterFromScenarioRef takes a scenario reference and returns
// a filter for it.
func WithScenarioFilterFromScenarioRef(ref *pb.Ref_Scenario) ScenarioFilterOpt {
	return func(f *ScenarioFilter) error {
		f.FromScenarioRef(ref)

		return nil
	}
}

// WithScenarioFilterFromSampleSubset takes a sample subset and returns a filter for it.
func WithScenarioFilterFromSampleSubset(subset *SampleSubset) ScenarioFilterOpt {
	return func(f *ScenarioFilter) error {
		return f.FromSampleSubset(subset)
	}
}

// ParseScenarioFilter takes command arguments that have been split by spaces
// and validates that they are composed of a valid scenario filter.
func ParseScenarioFilter(args []string) (*ScenarioFilter, error) {
	f, err := NewScenarioFilter()
	if err != nil {
		return nil, err
	}

	// No filter args means everything
	if len(args) == 0 || len(args) == 1 && args[0] == "" {
		f.SelectAll = true

		return f, nil
	}

	// Determine a name filer and/or variant vector for filtering
	for _, arg := range args {
		if !strings.Contains(arg, ":") {
			// It isn't a variant pair so it must be the name
			if f.Name != "" {
				// But we already have a name
				return f, fmt.Errorf("invalid variant filter: already found variant name %s and given another %s", f.Name, arg)
			}
			f.Name = arg

			continue
		}

		parts := strings.Split(arg, ":")
		if len(parts) != 2 {
			return f, fmt.Errorf("invalid variant filer (%s): filter must be a key:value pair", arg)
		}

		// Determine if it's an inclusive or exclusive filter
		if strings.HasPrefix(parts[0], "!") {
			// It's an exclude filter

			vec := NewVector()
			vec.Add(NewElement(strings.TrimPrefix(parts[0], "!"), parts[1]))
			ex, err := NewExclude(pb.Matrix_Exclude_MODE_CONTAINS, vec)
			if err != nil {
				return f, fmt.Errorf("invalid variant filter: %w", err)
			}
			f.Exclude = append(f.Exclude, ex)

			continue
		}

		// It's an include filter
		if f.Include == nil {
			f.Include = NewVector()
		}
		f.Include.Add(NewElement(parts[0], parts[1]))
	}

	return f, nil
}

// Proto returns the scenario filter as a proto filter.
func (sf *ScenarioFilter) Proto() *pb.Scenario_Filter {
	pbf := &pb.Scenario_Filter{
		Name: sf.Name,
	}
	if sf.Include != nil {
		pbf.Include = sf.Include.Proto()
	}

	if len(sf.Exclude) > 0 {
		pbf.Exclude = []*pb.Matrix_Exclude{}
		for _, e := range sf.Exclude {
			pbf.Exclude = append(pbf.GetExclude(), e.Proto())
		}
	}

	if sf.SelectAll {
		pbf.SelectAll = &pb.Scenario_Filter_SelectAll{}
	}

	if sf.IntersectionMatrix != nil {
		pbf.IntersectionMatrix = sf.IntersectionMatrix.Proto()
	}

	return pbf
}

// FromProto unmarshals a proto filter into itself.
func (sf *ScenarioFilter) FromProto(filter *pb.Scenario_Filter) {
	if filter == nil {
		return
	}

	sf.Name = filter.GetName()

	if i := filter.GetInclude(); i != nil {
		sf.Include = NewVectorFromProto(i)
	}

	if e := filter.GetExclude(); len(e) > 0 {
		sf.Exclude = []*Exclude{}
		for _, exp := range e {
			ex := &Exclude{}
			ex.FromProto(exp)
			sf.Exclude = append(sf.Exclude, ex)
		}
	}

	if sa := filter.GetSelectAll(); sa != nil {
		sf.SelectAll = true
	}

	if sim := filter.GetIntersectionMatrix(); sim != nil {
		nm := NewMatrix()
		nm.FromProto(sim)
		sf.IntersectionMatrix = nm
	}
}

// FromScenarioRef takes a reference to a scenario and returns a filter for it.
func (sf *ScenarioFilter) FromScenarioRef(ref *pb.Ref_Scenario) {
	sf.Name = ref.GetId().GetName()
	sf.Include = NewVectorFromProto(ref.GetId().GetVariants())
}

// FromSampleSubset takes a sample subset and returns a scenario filter for it.
func (sf *ScenarioFilter) FromSampleSubset(subset *SampleSubset) error {
	if sf == nil || subset == nil {
		return nil
	}

	// Sample subsets and scenario filters don't match 1:1, so we need to handle special cases
	// where a sample subset might have configuration that conflicts with a scenario filter.
	// These differences should be prevented during decoding time, but we'll still validate them here.
	if subset.Name == "" && subset.ScenarioName == "" && subset.ScenarioFilter == "" {
		return errors.New("cannot filter scenarios from subset, the subset does not include a scenario name")
	}

	if subset.ScenarioFilter != "" && subset.Matrix != nil && len(subset.Matrix.Vectors) > 0 {
		return errors.New("cannot filter scenarios from subset, only of matrix and scenario_filter can be set")
	}

	// Set the name. It's either our subset name, our scenario_name, or the scenario name in the filter.
	sf.Name = subset.Name
	if subset.ScenarioName != "" {
		sf.Name = subset.ScenarioName
	}

	// Set our matrix if we have one
	if subset.Matrix != nil && len(subset.Matrix.Vectors) > 0 {
		sf.IntersectionMatrix = subset.Matrix
	} else {
		sf.IntersectionMatrix = nil
	}

	// Handle select all. If we don't have a filter or a matrix we can assume we're selecting all.
	// We'll do that by making our filter just our name and parsing it.
	filter := subset.ScenarioFilter
	if sf.IntersectionMatrix == nil && filter == "" {
		filter = sf.Name
	}

	if filter != "" {
		psf, err := ParseScenarioFilter(strings.Split(filter, " "))
		if err != nil {
			return err
		}

		if psf.Name != "" {
			sf.Name = psf.Name
		}

		// Make sure we didn't set both a scenario_filter and scenario_name with conflicting names.
		if subset.ScenarioName != "" && psf.Name != "" && subset.ScenarioName != psf.Name {
			return fmt.Errorf("scenario_name '%s' and the scenario name in scenario_filter '%s' must match",
				subset.ScenarioName, psf.Name,
			)
		}

		sf.Include = psf.Include
		sf.Exclude = psf.Exclude
		sf.SelectAll = psf.SelectAll

		return nil
	}

	return nil
}

// ScenariosSelect takes a scenario filter and returns a slice of matching
// scenarios.
func (fp *FlightPlan) ScenariosSelect(f *ScenarioFilter) []*Scenario {
	if f == nil {
		return nil
	}

	if f.SelectAll {
		return fp.Scenarios()
	}

	scenarios := []*Scenario{}
	for _, s := range fp.Scenarios() {
		s := s
		if !s.Match(f) {
			continue
		}

		scenarios = append(scenarios, s)
	}

	return scenarios
}
