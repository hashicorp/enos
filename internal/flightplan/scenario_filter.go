package flightplan

import (
	"fmt"
	"strings"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ScenarioFilter is a filter for scenarios
type ScenarioFilter struct {
	Name      string
	Include   *Vector
	Exclude   []*Exclude
	SelectAll bool
}

// String returns the scenario filter as a string
func (sf *ScenarioFilter) String() string {
	if sf == nil {
		return ""
	}

	str := sf.Name

	if sf.SelectAll {
		return str
	}

	for _, i := range sf.Include.unordered {
		str = fmt.Sprintf("%s %s", str, i.String())
	}

	for _, e := range sf.Exclude {
		for _, elm := range e.Vector.unordered {
			str = fmt.Sprintf("%s !%s:%s", str, elm.Key, elm.Val)
		}
	}

	return str
}

// ScenarioFilterOpt is a scenario filter constructor functional option
type ScenarioFilterOpt func(*ScenarioFilter) error

// NewScenarioFilter takes in options and returns a new filter
func NewScenarioFilter(opts ...ScenarioFilterOpt) (*ScenarioFilter, error) {
	f := &ScenarioFilter{
		Include: NewVector(),
	}

	for _, opt := range opts {
		err := opt(f)
		if err != nil {
			return f, err
		}
	}

	return f, nil
}

// WithScenarioFilterName sets the filter name
func WithScenarioFilterName(name string) ScenarioFilterOpt {
	return func(f *ScenarioFilter) error {
		f.Name = name
		return nil
	}
}

// WithScenarioFilterSelectAll makes the filter select all
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

// WithScenarioFilterParse parses the given filter
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

// WithScenarioFilterDecode decodes a filter from a proto Filter
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

// ParseScenarioFilter takes command arguments that have been split by spaces
// and validates that they are composed of a valid scenario filter.
func ParseScenarioFilter(args []string) (*ScenarioFilter, error) {
	f, err := NewScenarioFilter()
	if err != nil {
		return nil, err
	}

	// No filter args means everything
	if len(args) == 0 {
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
			ex, err := NewExclude(pb.Scenario_Filter_Exclude_MODE_CONTAINS, vec)
			if err != nil {
				return f, fmt.Errorf("invalid variant filter: %w", err)
			}
			f.Exclude = append(f.Exclude, ex)
			continue
		}

		// It's an include filter
		f.Include.Add(NewElement(parts[0], parts[1]))
	}

	return f, nil
}

// Proto returns the scenario filter as a proto filter
func (sf *ScenarioFilter) Proto() *pb.Scenario_Filter {
	pbf := &pb.Scenario_Filter{
		Name:    sf.Name,
		Include: sf.Include.Proto(),
	}

	if len(sf.Exclude) > 0 {
		pbf.Exclude = []*pb.Scenario_Filter_Exclude{}
		for _, e := range sf.Exclude {
			pbf.Exclude = append(pbf.Exclude, e.Proto())
		}
	}

	if sf.SelectAll {
		pbf.SelectAll = &pb.Scenario_Filter_SelectAll{}
	}

	return pbf
}

// FromProto unmarshals a proto filter into itself
func (sf *ScenarioFilter) FromProto(filter *pb.Scenario_Filter) {
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
}

// FromScenarioRef takes a reference to a scenario and returns a filter for it
func (sf *ScenarioFilter) FromScenarioRef(ref *pb.Ref_Scenario) {
	sf.Name = ref.GetId().GetName()
	sf.Include = NewVectorFromProto(ref.GetId().GetVariants())
}

// ScenariosSelect takes a scenario filter and returns a slice of matching
// scenarios.
func (fp *FlightPlan) ScenariosSelect(f *ScenarioFilter) []*Scenario {
	if f.SelectAll {
		return fp.Scenarios
	}

	scenarios := []*Scenario{}
	for _, s := range fp.Scenarios {
		if !s.Match(f) {
			continue
		}

		scenarios = append(scenarios, s)
	}

	return scenarios
}
