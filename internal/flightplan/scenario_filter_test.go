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
		{Name: "fresh-install", Variants: Vector{NewElement("backend", "raft"), NewElement("arch", "arm64")}},
		{Name: "fresh-install", Variants: Vector{NewElement("backend", "raft"), NewElement("arch", "amd64")}},
		{Name: "fresh-install", Variants: Vector{NewElement("backend", "consul"), NewElement("arch", "arm64")}},
		{Name: "fresh-install", Variants: Vector{NewElement("backend", "consul"), NewElement("arch", "amd64")}},
		{Name: "upgrade", Variants: Vector{NewElement("backend", "raft"), NewElement("arch", "arm64")}},
		{Name: "upgrade", Variants: Vector{NewElement("backend", "raft"), NewElement("arch", "amd64")}},
		{Name: "upgrade", Variants: Vector{NewElement("backend", "consul"), NewElement("arch", "arm64")}},
		{Name: "upgrade", Variants: Vector{NewElement("backend", "consul"), NewElement("arch", "amd64")}},
	}

	for _, test := range []struct {
		desc      string
		scenarios []*Scenario
		filter    *ScenarioFilter
		expected  []*Scenario
	}{
		{
			"name only",
			scenarios,
			&ScenarioFilter{Name: "upgrade"},
			scenarios[4:],
		},
		{
			"name no match",
			scenarios,
			&ScenarioFilter{Name: "package"},
			[]*Scenario{},
		},
		{
			"select all",
			scenarios,
			&ScenarioFilter{SelectAll: true},
			scenarios,
		},
		{
			"variant with no name",
			scenarios,
			&ScenarioFilter{
				Include: Vector{NewElement("backend", "consul")},
				Exclude: []*Exclude{
					{ExcludeMatch, Vector{NewElement("arch", "arm64")}},
				},
			},
			[]*Scenario{scenarios[3], scenarios[7]},
		},
		{
			"variant with name",
			scenarios,
			&ScenarioFilter{
				Name:    "upgrade",
				Include: Vector{NewElement("backend", "raft")},
				Exclude: []*Exclude{
					{ExcludeMatch, Vector{NewElement("arch", "amd64")}},
				},
			},
			[]*Scenario{scenarios[4]},
		},
		{
			"variant matches vector but requires more",
			scenarios,
			&ScenarioFilter{
				Name:    "upgrade",
				Include: Vector{NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("edition", "ent")},
			},
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
		filterArg []string
		expected  *ScenarioFilter
	}{
		{
			"blank filter",
			[]string{},
			&ScenarioFilter{SelectAll: true},
		},
		{
			"filter with only name",
			[]string{"test"},
			&ScenarioFilter{Name: "test"},
		},
		{
			"filter with name and variants",
			[]string{"test", "backend:consul", "!arch:arm64"},
			&ScenarioFilter{
				Name:    "test",
				Include: Vector{NewElement("backend", "consul")},
				Exclude: []*Exclude{
					{ExcludeMatch, Vector{NewElement("arch", "arm64")}},
				},
			},
		},
		{
			"filter with no name and variants",
			[]string{"!arch:amd64", "backend:raft"},
			&ScenarioFilter{
				Include: Vector{NewElement("backend", "raft")},
				Exclude: []*Exclude{
					{ExcludeMatch, Vector{NewElement("arch", "amd64")}},
				},
			},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			filter, err := NewScenarioFilter(WithScenarioFilterParse(test.filterArg))
			require.NoError(t, err)
			require.EqualValues(t, test.expected, filter)
		})
	}
}
