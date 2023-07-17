package flightplan

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// Test_ScenarioFilter_WithScenarioFilterFromScenarioRef tests filtering a
// scenario that was created from a scenario reference.
func Test_ScenarioFilter_WithScenarioFilterFromScenarioRef(t *testing.T) {
	t.Parallel()

	ref := &pb.Ref_Scenario{
		Id: &pb.Scenario_ID{
			Name: "foo",
			Variants: &pb.Scenario_Filter_Vector{
				Elements: []*pb.Scenario_Filter_Element{
					{Key: "backend", Value: "raft"},
					{Key: "cloud", Value: "aws"},
				},
			},
		},
	}
	expected := &ScenarioFilter{
		Name: "foo",
		Include: &Vector{elements: []Element{
			NewElement("backend", "raft"),
			NewElement("cloud", "aws"),
		}},
	}
	sf, err := NewScenarioFilter(WithScenarioFilterFromScenarioRef(ref))
	require.NoError(t, err)
	require.Equal(t, expected.Name, sf.Name)
	require.Len(t, sf.Include.elements, 2)
	require.Equal(t, expected.Include.elements[0].Key, sf.Include.elements[0].Key)
	require.Equal(t, expected.Include.elements[0].Val, sf.Include.elements[0].Val)
	require.Equal(t, expected.Include.elements[1].Key, sf.Include.elements[1].Key)
	require.Equal(t, expected.Include.elements[1].Val, sf.Include.elements[1].Val)
}

// Test_ScenarioFilter_ScenariosSelect tests that a flight plan returns the
// scenarios when selecting with a filter.
func Test_ScenarioFilter_ScenariosSelect(t *testing.T) {
	t.Parallel()

	scenarios := []*Scenario{
		{Name: "fresh-install", Variants: &Vector{elements: []Element{NewElement("backend", "raft"), NewElement("arch", "arm64")}}},
		{Name: "fresh-install", Variants: &Vector{elements: []Element{NewElement("backend", "raft"), NewElement("arch", "amd64")}}},
		{Name: "fresh-install", Variants: &Vector{elements: []Element{NewElement("backend", "consul"), NewElement("arch", "arm64")}}},
		{Name: "fresh-install", Variants: &Vector{elements: []Element{NewElement("backend", "consul"), NewElement("arch", "amd64")}}},
		{Name: "upgrade", Variants: &Vector{elements: []Element{NewElement("backend", "raft"), NewElement("arch", "arm64")}}},
		{Name: "upgrade", Variants: &Vector{elements: []Element{NewElement("backend", "raft"), NewElement("arch", "amd64")}}},
		{Name: "upgrade", Variants: &Vector{elements: []Element{NewElement("backend", "consul"), NewElement("arch", "arm64")}}},
		{Name: "upgrade", Variants: &Vector{elements: []Element{NewElement("backend", "consul"), NewElement("arch", "amd64")}}},
		{Name: "no-variant"},
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
			scenarios[4:8],
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
				Include: &Vector{elements: []Element{NewElement("backend", "consul")}},
				Exclude: []*Exclude{
					{pb.Scenario_Filter_Exclude_MODE_CONTAINS, &Vector{elements: []Element{NewElement("arch", "arm64")}}},
				},
			},
			[]*Scenario{scenarios[3], scenarios[7]},
		},
		{
			"variant with name",
			scenarios,
			&ScenarioFilter{
				Name:    "upgrade",
				Include: &Vector{elements: []Element{NewElement("backend", "raft")}},
				Exclude: []*Exclude{
					{pb.Scenario_Filter_Exclude_MODE_CONTAINS, &Vector{elements: []Element{NewElement("arch", "amd64")}}},
				},
			},
			[]*Scenario{scenarios[4]},
		},
		{
			"variant matches vector but requires more",
			scenarios,
			&ScenarioFilter{
				Name:    "upgrade",
				Include: &Vector{elements: []Element{NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("edition", "ent")}},
			},
			[]*Scenario{},
		},
		{
			"variant filter pass to scenario without variants",
			scenarios,
			&ScenarioFilter{
				Name:    "no-variant",
				Include: &Vector{elements: []Element{NewElement("backend", "raft")}},
			},
			[]*Scenario{},
		},
	} {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

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
			&ScenarioFilter{
				SelectAll: true,
				Include:   &Vector{},
			},
		},
		{
			"filter with only name",
			[]string{"test"},
			&ScenarioFilter{
				Name:    "test",
				Include: &Vector{},
			},
		},
		{
			"filter with name and variants",
			[]string{"test", "backend:consul", "!arch:arm64"},
			&ScenarioFilter{
				Name:    "test",
				Include: &Vector{elements: []Element{NewElement("backend", "consul")}},
				Exclude: []*Exclude{
					{pb.Scenario_Filter_Exclude_MODE_CONTAINS, &Vector{elements: []Element{NewElement("arch", "arm64")}}},
				},
			},
		},
		{
			"filter with no name and variants",
			[]string{"!arch:amd64", "backend:raft"},
			&ScenarioFilter{
				Include: &Vector{elements: []Element{NewElement("backend", "raft")}},
				Exclude: []*Exclude{
					{pb.Scenario_Filter_Exclude_MODE_CONTAINS, &Vector{elements: []Element{NewElement("arch", "amd64")}}},
				},
			},
		},
	} {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			filter, err := NewScenarioFilter(WithScenarioFilterParse(test.filterArg))
			require.NoError(t, err)
			require.EqualValues(t, test.expected, filter)
		})
	}
}
