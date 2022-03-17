package flightplan

import (
	"fmt"
	"strings"
)

// ScenarioFilter is a filter for scenarios
type ScenarioFilter struct {
	Name      string
	Include   Vector
	Exclude   []*Exclude
	SelectAll bool
}

// ScenarioFilterOpt is a scenario filter constructor functional option
type ScenarioFilterOpt func(*ScenarioFilter) error

// NewScenarioFilter takes in options and returns a new filter
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
func WithScenarioFilterMatchingVariants(vec Vector) ScenarioFilterOpt {
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

// ParseScenarioFilter takes command arguments that have been split by spaces
// and validates that they are composed of a valid scenario filter.
func ParseScenarioFilter(args []string) (*ScenarioFilter, error) {
	f := &ScenarioFilter{}

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
			ex, err := NewExclude(ExcludeMatch, Vector{
				NewElement(strings.TrimPrefix(parts[0], "!"), parts[1]),
			})
			if err != nil {
				return f, fmt.Errorf("invalid variant filter: %w", err)
			}
			f.Exclude = append(f.Exclude, ex)
			continue
		}

		// It's an include filter
		f.Include = append(f.Include, NewElement(parts[0], parts[1]))
	}

	return f, nil
}

// ScenariosSelect takes a scenario filter and returns a slice of matching
// scenarios.
func (fp *FlightPlan) ScenariosSelect(f *ScenarioFilter) []*Scenario {
	if f.SelectAll {
		return fp.Scenarios
	}

	scenarios := []*Scenario{}
	for _, s := range fp.Scenarios {
		// Get scenarios that match our name
		if f.Name != "" && f.Name != s.Name {
			// Our name doesn't match the filter name
			continue
		}

		// Make sure it matches any includes
		if len(f.Include) > 0 {
			if !s.Variants.ContainsValues(f.Include) {
				// Our scenario variants don't include all of the required elements
				continue
			}
		}

		skip := false
		for _, ex := range f.Exclude {
			if ex.ExcludeVector(s.Variants) {
				skip = true
				break
			}
		}
		if skip {
			// We matched an exclusion
			continue
		}

		scenarios = append(scenarios, s)
	}

	return scenarios
}
