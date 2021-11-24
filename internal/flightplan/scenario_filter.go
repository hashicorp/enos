package flightplan

import "strings"

// ScenarioFilter is a filter for scenarios
type ScenarioFilter struct {
	Name      string
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

// WithScenarioFilterParse parses the given filter
func WithScenarioFilterParse(filter string) ScenarioFilterOpt {
	return func(f *ScenarioFilter) error {
		if filter == "" {
			f.SelectAll = true
			return nil
		}

		// NOTE: We only support filtering by names. When we add variants
		// to scenario filters we'll need to do a lot more here.
		parts := strings.Split(filter, " ")
		f.Name = parts[0]
		return nil
	}
}

// ScenariosSelect takes a scenario filter and returns a slice of matching
// scenarios.
func (fp *FlightPlan) ScenariosSelect(f *ScenarioFilter) []*Scenario {
	if f.SelectAll {
		return fp.Scenarios
	}

	scenarios := []*Scenario{}
	for _, s := range fp.Scenarios {
		if s.Name == f.Name {
			scenarios = append(scenarios, s)
		}
	}

	return scenarios
}
