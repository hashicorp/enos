package flightplan

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test_ScenarioFilter_ScenariosSelect tests that a flight plan returns the
// scenarios when selecting with a filter.
func Test_ScenarioFilter_ScenariosSelect(t *testing.T) {
	t.Parallel()

	scenarios := []*Scenario{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	}

	for _, test := range []struct {
		desc      string
		scenarios []*Scenario
		filter    *ScenarioFilter
		expected  []*Scenario
	}{
		{
			"match one",
			scenarios,
			&ScenarioFilter{Name: "b"},
			[]*Scenario{scenarios[1]},
		},
		{
			"match all",
			scenarios,
			&ScenarioFilter{SelectAll: true},
			scenarios,
		},
		{
			"match none",
			scenarios,
			&ScenarioFilter{Name: "nope"},
			[]*Scenario{},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			fp := &FlightPlan{
				Scenarios: test.scenarios,
			}
			require.EqualValues(t, test.expected, fp.ScenariosSelect(test.filter))
		})
	}
}

// Test_ScenarioFilter_Parse tests that when the given string is parsed that
// and expected filter is returned.
func Test_ScenarioFilter_Parse(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		desc      string
		filterArg string
		expected  *ScenarioFilter
	}{
		{
			"blank filter",
			"",
			&ScenarioFilter{SelectAll: true},
		},
		{
			"filter with only name",
			"test",
			&ScenarioFilter{Name: "test"},
		},
		{
			"filter with name and variants",
			"test variant:foo",
			&ScenarioFilter{Name: "test"},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			filter, err := NewScenarioFilter(WithScenarioFilterParse(test.filterArg))
			require.NoError(t, err)
			require.EqualValues(t, test.expected, filter)
		})
	}
}
