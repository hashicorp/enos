package flightplan

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// Test_Decode_Scenario_Matrix tests decoding of a matrix in scenarios.
func Test_Decode_Scenario_Matrix(t *testing.T) {
	t.Parallel()

	modulePath, err := filepath.Abs("./tests/simple_module")
	require.NoError(t, err)

	for desc, test := range map[string]struct {
		hcl      string
		expected *FlightPlan
		fail     bool
	}{
		"matrix with label": {
			fail: true,
			hcl: fmt.Sprintf(`
			module "backend" {
			  source = "%s"
			}

			scenario "backend" {
			  matrix "something" {
			    cathat = ["thing1", "thing2"]
			  }

			  step "first" {
			    module = module.backend
			  }
			}
			`, modulePath),
		},
		"more than one matrix block": {
			fail: true,
			hcl: fmt.Sprintf(`
			module "backend" {
			  source = "%s"
			}

			scenario "backend" {
			  matrix {
			    cathat = ["thing1", "thing2"]
			  }

			  matrix {
			    onefish = ["redfish", "bluefish"]
			  }

			  step "first" {
			    module = module.backend
			  }
			}
			`, modulePath),
		},
		"invalid matrix value": {
			fail: true,
			hcl: fmt.Sprintf(`
			module "backend" {
			  source = "%s"
			}

			scenario "backend" {
			  matrix {
			    onefish = "redfish"
			  }

			  step "first" {
			    module = module.backend
			  }
			}
			`, modulePath),
		},
		"invalid block": {
			fail: true,
			hcl: fmt.Sprintf(`
			module "backend" {
			  source = "%s"
			}

			scenario "backend" {
			  matrix {
			    anotherthing {
			    }
			  }

			  step "first" {
			    module = module.backend
			  }
			}
			`, modulePath),
		},
		"invalid value include": {
			fail: true,
			hcl: fmt.Sprintf(`
			module "backend" {
			  source = "%s"
			}

			scenario "backend" {
			  matrix {
			    include {
			      onefish = "redfish"
			    }
			  }

			  step "first" {
			    module = module.backend
			  }
			}
			`, modulePath),
		},
		"invalid block include": {
			fail: true,
			hcl: fmt.Sprintf(`
			module "backend" {
			  source = "%s"
			}

			scenario "backend" {
			  matrix {
			    include {
			      something {
			      }
			    }
			  }

			  step "first" {
			    module = module.backend
			  }
			}
			`, modulePath),
		},
		"invalid value exclude": {
			fail: true,
			hcl: fmt.Sprintf(`
			module "backend" {
			  source = "%s"
			}

			scenario "backend" {
			  matrix {
			    exclude {
			      onefish = "redfish"
			    }
			  }

			  step "first" {
			    module = module.backend
			  }
			}
			`, modulePath),
		},
		"invalid block exclude": {
			fail: true,
			hcl: fmt.Sprintf(`
			module "backend" {
			  source = "%s"
			}

			scenario "backend" {
			  matrix {
			    exclude {
			      something {
			      }
			    }
			  }

			  step "first" {
			    module = module.backend
			  }
			}
			`, modulePath),
		},
		"valid matrix": {
			hcl: fmt.Sprintf(`
module "books" {
  source = "%s"
}

scenario "nighttime" {
  matrix {
    cathat  = ["thing1", "thing2"]
	onefish = ["redfish", "bluefish"]

	include {
	  cathat  = ["sally", "conrad"]
	  onefish = ["twofish"]
	}

	exclude {
	  cathat  = ["thing1"]
	  onefish = ["redfish", "bluefish"]
	}
  }

  step "read" {
    module = module.books

	variables {
	  cathat  = matrix.cathat
	  onefish = matrix.onefish
	}
  }
}
`, modulePath),
			expected: &FlightPlan{
				TerraformCLIs: []*TerraformCLI{
					DefaultTerraformCLI(),
				},
				Modules: []*Module{
					{
						Name:   "books",
						Source: modulePath,
					},
				},
				ScenarioBlocks: DecodedScenarioBlocks{
					{
						Name: "nighttime",
						Scenarios: []*Scenario{
							{
								Name:         "nighttime",
								Variants:     NewVector(NewElement("cathat", "conrad"), NewElement("onefish", "twofish")),
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name: "read",
										Module: &Module{
											Name:   "books",
											Source: modulePath,
											Attrs: map[string]cty.Value{
												"cathat":  testMakeStepVarValue(cty.StringVal("conrad")),
												"onefish": testMakeStepVarValue(cty.StringVal("twofish")),
											},
										},
									},
								},
							},
							{
								Name:         "nighttime",
								Variants:     NewVector(NewElement("cathat", "sally"), NewElement("onefish", "twofish")),
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name: "read",
										Module: &Module{
											Name:   "books",
											Source: modulePath,
											Attrs: map[string]cty.Value{
												"cathat":  testMakeStepVarValue(cty.StringVal("sally")),
												"onefish": testMakeStepVarValue(cty.StringVal("twofish")),
											},
										},
									},
								},
							},
							{
								Name:         "nighttime",
								Variants:     NewVector(NewElement("cathat", "thing2"), NewElement("onefish", "bluefish")),
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name: "read",
										Module: &Module{
											Name:   "books",
											Source: modulePath,
											Attrs: map[string]cty.Value{
												"cathat":  testMakeStepVarValue(cty.StringVal("thing2")),
												"onefish": testMakeStepVarValue(cty.StringVal("bluefish")),
											},
										},
									},
								},
							},
							{
								Name:         "nighttime",
								Variants:     NewVector(NewElement("cathat", "thing2"), NewElement("onefish", "redfish")),
								TerraformCLI: DefaultTerraformCLI(),
								Steps: []*ScenarioStep{
									{
										Name: "read",
										Module: &Module{
											Name:   "books",
											Source: modulePath,
											Attrs: map[string]cty.Value{
												"cathat":  testMakeStepVarValue(cty.StringVal("thing2")),
												"onefish": testMakeStepVarValue(cty.StringVal("redfish")),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			fp, err := testDecodeHCL(t, []byte(test.hcl), DecodeTargetAll)
			if test.fail {
				require.Error(t, err)

				return
			}
			require.NoError(t, err)
			testRequireEqualFP(t, fp, test.expected)
		})
	}
}

func Test_Matrix_Vector_Equal(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in    *Vector
		other *Vector
		equal bool
	}{
		"equal": {
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			true,
		},
		"ordered but unequal Elements": {
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			false,
		},
		"equal Values": {
			NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			false,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.equal, test.in.Equal(test.other))
		})
	}
}

func Test_Matrix_Vector_ContainsUnordered(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in    *Vector
		other *Vector
		match bool
	}{
		"exact": {
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			true,
		},
		"unordered unequal len match": {
			NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")),
			false,
		},
		"unordered exact": {
			NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			true,
		},
		"equal len no match": {
			NewVector(NewElement("backend", "myssql"), NewElement("backend", "raft")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			false,
		},
		"unequal len no match": {
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")),
			false,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.match, test.in.ContainsUnordered(test.other))
		})
	}
}

func Test_Matrix_Vector_EqualUnordered(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in    *Vector
		other *Vector
		equal bool
	}{
		"equal": {
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			true,
		},
		"ordered but unequal Elements": {
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			false,
		},
		"equal Values": {
			NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
			NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
			true,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.equal, test.in.EqualUnordered(test.other))
		})
	}
}

func Test_Matrix_CartesianProduct(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       *Matrix
		expected *Matrix
	}{
		"nil vectors": {
			&Matrix{Vectors: nil},
			&Matrix{Vectors: nil},
		},
		"empty vectors": {
			&Matrix{Vectors: []*Vector{}},
			&Matrix{Vectors: nil},
		},
		"regular": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64")),
				NewVector(NewElement("distro", "ubuntu"), NewElement("distro", "rhel")),
			}},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("distro", "rhel")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64"), NewElement("distro", "rhel")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64"), NewElement("distro", "rhel")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64"), NewElement("distro", "rhel")),
			}},
		},
		"irregular": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64"), NewElement("arch", "ppc64")),
				NewVector(NewElement("distro", "ubuntu")),
				NewVector(NewElement("test", "fresh-install"), NewElement("test", "upgrade"), NewElement("test", "security")),
			}},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "security")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "security")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "security")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "security")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "security")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "security")),
			}},
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected.Vectors, test.in.CartesianProduct().Vectors)
		})
	}
}

func Test_Matrix_CartesianProduct_empty_vector(t *testing.T) {
	t.Parallel()

	m := NewMatrix()
	m.AddVector(NewVector())
	m.AddVector(NewVector())

	require.Equal(t, &Matrix{}, m.CartesianProduct())
}

func Test_Matrix_UniqueValues(t *testing.T) {
	t.Parallel()

	m1 := NewMatrix()
	m1.AddVector(NewVector(NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")))
	m1.AddVector(NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")))
	m1.AddVector(NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")))
	m1.AddVector(NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64")))
	m1.AddVector(NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64"), NewElement("arch", "ppc64")))
	m1.AddVector(NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")))

	m2 := NewMatrix()
	m2.AddVector(NewVector(NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")))
	m2.AddVector(NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")))
	m2.AddVector(NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64")))
	m2.AddVector(NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64"), NewElement("arch", "ppc64")))

	uniq := m1.UniqueValues()
	require.Len(t, uniq.Vectors, len(m2.Vectors))

	for i := range m2.Vectors {
		require.EqualValues(t, m2.Vectors[i].elements, uniq.Vectors[i].elements)
	}
}

func Test_Matrix_HasVectorSorted(t *testing.T) {
	t.Parallel()

	m1 := NewMatrix()
	m1.AddVector(NewVector(NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")))
	m1.AddVector(NewVector(NewElement("backend", "mysql"), NewElement("backend", "postgres"), NewElement("backend", "mssql")))
	m1.AddVector(NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")))
	m1.AddVector(NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")))
	m1.AddVector(NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64")))
	m1.AddVector(NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64"), NewElement("arch", "ppc64")))
	m1.AddVector(NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")))

	m1.Sort()
	for desc, test := range map[string]struct {
		has bool
		vec *Vector
	}{
		"not there": {
			has: false,
			vec: NewVector(NewElement("arch", "s390x"), NewElement("arch", "arm64")),
		},
		"partial ordered": {
			has: false,
			vec: NewVector(NewElement("backend", "mysql"), NewElement("backend", "postgres")),
		},
		"unorderd": {
			has: true,
			vec: NewVector(NewElement("backend", "mssql"), NewElement("backend", "consul"), NewElement("backend", "raft")),
		},
		"in order": {
			has: true,
			vec: NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64"), NewElement("arch", "ppc64")),
		},
	} {
		desc := desc
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			test.vec.Sort()
			has := m1.HasVectorSorted(test.vec)
			require.Equal(t, test.has, has)
		})
	}
}

func Test_Matrix_Proto_RoundTrip(t *testing.T) {
	t.Parallel()
	expected := &Matrix{Vectors: []*Vector{
		NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
		NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
		NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
		NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
	}}

	got := NewMatrix()
	got.FromProto(expected.Copy().Proto())
	require.EqualValues(t, expected, got)
}

func Test_Matrix_Unique(t *testing.T) {
	t.Parallel()

	m1 := NewMatrix()
	m1.AddVector(NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")))
	m1.AddVector(NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")))
	m1.AddVector(NewVector(NewElement("backend", "consul"), NewElement("backend", "raft"), NewElement("backend", "myssql")))
	m1.AddVector(NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")))
	m1.AddVector(NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")))
	m1.AddVector(NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64")))
	m1.AddVector(NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64")))
	m1.AddVector(NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")))

	m2 := NewMatrix()
	m2.AddVector(NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")))
	m2.AddVector(NewVector(NewElement("backend", "consul"), NewElement("backend", "raft"), NewElement("backend", "myssql")))
	m2.AddVector(NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")))
	m2.AddVector(NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64")))
	m2.AddVector(NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")))

	require.Equal(t, m2, m1.Unique())
}

func Test_Matrix_Filter_Filter_Parse(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in        *Matrix
		filterStr string
		expected  *Matrix
	}{
		"all": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
			"",
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
		},
		"include-some": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
			"backend:raft",
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
			}},
		},
		"include-one": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
			"backend:raft arch:amd64",
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
		},
		"exclude-one": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
			"!arch:amd64",
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
		},
		"exclude-some": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
			"!arch:amd64 !backend:consul",
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
			}},
		},
		"include-and-exclude-one": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
			"!arch:amd64 backend:raft",
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
			}},
		},
		"include-and-exclude-some": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
			"!arch:amd64 !backend:raft arch:aarch64",
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			f, err := NewScenarioFilter(WithScenarioFilterParse(strings.Split(test.filterStr, " ")))
			require.NoError(t, err)
			require.True(t, test.expected.Equal(test.in.Filter(f)))
		})
	}
}

func Test_Matrix_Filter_ScenarioFilter(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       *Matrix
		filter   *ScenarioFilter
		expected *Matrix
	}{
		"all": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
			&ScenarioFilter{SelectAll: true},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
		},
		"include-some": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
			&ScenarioFilter{
				Include: NewVector(NewElement("backend", "raft")),
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
			}},
		},
		"include-one": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
			&ScenarioFilter{
				Include: NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
		},
		"exclude-one": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
			&ScenarioFilter{
				Exclude: []*Exclude{
					{
						Mode:   pb.Matrix_Exclude_MODE_CONTAINS,
						Vector: NewVector(NewElement("backend", "raft")),
					},
				},
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
		},
		"exclude-some": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
			}},
			&ScenarioFilter{
				Exclude: []*Exclude{
					{
						Mode:   pb.Matrix_Exclude_MODE_CONTAINS,
						Vector: NewVector(NewElement("arch", "amd64")),
					},
					{
						Mode:   pb.Matrix_Exclude_MODE_CONTAINS,
						Vector: NewVector(NewElement("backend", "consul")),
					},
				},
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
			}},
		},
		"include-and-exclude-one": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
			&ScenarioFilter{
				Include: NewVector(NewElement("backend", "raft")),
				Exclude: []*Exclude{
					{
						Mode:   pb.Matrix_Exclude_MODE_CONTAINS,
						Vector: NewVector(NewElement("arch", "amd64")),
					},
				},
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
			}},
		},
		"include-and-exclude-some": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
			&ScenarioFilter{
				Include: NewVector(NewElement("arch", "aarch64")),
				Exclude: []*Exclude{
					{
						Mode:   pb.Matrix_Exclude_MODE_CONTAINS,
						Vector: NewVector(NewElement("arch", "amd64")),
					},
					{
						Mode:   pb.Matrix_Exclude_MODE_CONTAINS,
						Vector: NewVector(NewElement("backend", "raft")),
					},
				},
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
		},
		"intersection-matrix": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
			&ScenarioFilter{
				IntersectionMatrix: &Matrix{Vectors: []*Vector{
					NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
					NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
					NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
					NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
				}},
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
		},
		"intersection-matrix-with-exclude": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
			&ScenarioFilter{
				Exclude: []*Exclude{
					{
						Mode:   pb.Matrix_Exclude_MODE_CONTAINS,
						Vector: NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
					},
				},
				IntersectionMatrix: &Matrix{Vectors: []*Vector{
					NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
					NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
					NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
					NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
					NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
					NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
				}},
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
		},
		"intersection-matrix-with-include": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "aarch64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
			}},
			&ScenarioFilter{
				Include: NewVector(NewElement("backend", "raft")),
				IntersectionMatrix: &Matrix{Vectors: []*Vector{
					NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
					NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
					NewVector(NewElement("backend", "consul"), NewElement("arch", "amd64")),
					NewVector(NewElement("backend", "consul"), NewElement("arch", "arm64")),
					NewVector(NewElement("backend", "consul"), NewElement("arch", "aarch64")),
				}},
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
			}},
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			require.True(t, test.expected.EqualUnordered(test.in.Filter(test.filter)))
		})
	}
}

func Test_Matrix_Equal(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       *Matrix
		other    *Matrix
		expected bool
	}{
		"both-nil": {
			in:       new(Matrix),
			other:    new(Matrix),
			expected: true,
		},
		"in-nil": {
			in: new(Matrix),
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			expected: false,
		},
		"other-nil": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other:    new(Matrix),
			expected: false,
		},
		"unbalanced-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: false,
		},
		"balanced-different-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("howdy", "partner"), NewElement("hey", "pal")),
				NewVector(NewElement("howdy", "friend"), NewElement("hey", "guy")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: false,
		},
		"unordered-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: false,
		},
		"equal": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			expected: true,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.expected, test.in.Equal(test.other))
		})
	}
}

func Test_Matrix_EqualUnordered(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       *Matrix
		other    *Matrix
		expected bool
	}{
		"both-nil": {
			in:       new(Matrix),
			other:    new(Matrix),
			expected: true,
		},
		"in-nil": {
			in: new(Matrix),
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			expected: false,
		},
		"other-nil": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other:    new(Matrix),
			expected: false,
		},
		"unbalanced-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: false,
		},
		"balanced-different-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("howdy", "partner"), NewElement("hey", "pal")),
				NewVector(NewElement("howdy", "friend"), NewElement("hey", "guy")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: false,
		},
		"unordered-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: true,
		},
		"unordered-elements-and-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("arch", "arm32"), NewElement("backend", "raft")),
				NewVector(NewElement("arch", "amd64"), NewElement("backend", "raft")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: true,
		},
		"equal": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			expected: true,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.expected, test.in.EqualUnordered(test.other))
		})
	}
}

func Test_Matrix_Exclude(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       *Matrix
		Excludes []*Exclude
		expected *Matrix
	}{
		"nil": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")),
			}},
			nil,
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")),
			}},
		},
		"empty": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")),
			}},
			[]*Exclude{},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")),
			}},
		},
		"exact": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")),
			}},
			[]*Exclude{
				{
					Mode:   pb.Matrix_Exclude_MODE_EXACTLY,
					Vector: NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				},
				{
					Mode:   pb.Matrix_Exclude_MODE_EXACTLY,
					Vector: NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")),
				},
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")),
			}},
		},
		"equal values": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				NewVector(NewElement("backend", "consul"), NewElement("backend", "raft")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")),
			}},
			[]*Exclude{
				{
					Mode:   pb.Matrix_Exclude_MODE_EQUAL_UNORDERED,
					Vector: NewVector(NewElement("backend", "raft"), NewElement("backend", "consul")),
				},
				{
					Mode:   pb.Matrix_Exclude_MODE_EQUAL_UNORDERED,
					Vector: NewVector(NewElement("arch", "arm64"), NewElement("arch", "amd64")),
				},
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")),
			}},
		},
		"match": {
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")),
				NewVector(NewElement("backend", "consul"), NewElement("backend", "raft"), NewElement("backend", "mysql")),
				NewVector(NewElement("backend", "raft"), NewElement("backend", "mysql"), NewElement("backend", "mssql")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "arm32")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("arch", "arm32")),
			}},
			[]*Exclude{
				{
					Mode:   pb.Matrix_Exclude_MODE_CONTAINS,
					Vector: NewVector(NewElement("backend", "mysql")),
				},
				{
					Mode:   pb.Matrix_Exclude_MODE_CONTAINS,
					Vector: NewVector(NewElement("arch", "arm64"), NewElement("arch", "arm32")),
				},
			},
			&Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")),
				NewVector(NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm64")),
			}},
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.expected.Vectors, test.in.Exclude(test.Excludes...).Vectors)
		})
	}
}

func Test_Matrix_IntersectionUnordered(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       *Matrix
		other    *Matrix
		expected *Matrix
	}{
		"both-nil": {
			in:       nil,
			other:    nil,
			expected: nil,
		},
		"in-nil": {
			in: nil,
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			expected: nil,
		},
		"other-nil": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other:    nil,
			expected: nil,
		},
		"unbalanced-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
		},
		"balanced-different-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("howdy", "partner"), NewElement("hey", "pal")),
				NewVector(NewElement("howdy", "friend"), NewElement("hey", "guy")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: nil,
		},
		"unordered-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
		},
		"unordered-elements-and-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("arch", "arm32"), NewElement("backend", "raft")),
				NewVector(NewElement("arch", "amd64"), NewElement("backend", "raft")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
		},
		"equal": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			expected: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			res := test.in.IntersectionContainsUnordered(test.other)
			require.Truef(t, test.expected.EqualUnordered(res), test.in.SymmetricDifferenceUnordered(res).String())
		})
	}
}

func Test_Matrix_SymmetricDifferenceUnordered(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		in       *Matrix
		other    *Matrix
		expected *Matrix
	}{
		"both-nil": {
			in:       nil,
			other:    nil,
			expected: nil,
		},
		"in-nil": {
			in: nil,
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			expected: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
		},
		"other-nil": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: nil,
			expected: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
		},
		"unbalanced-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
		},
		"balanced-different-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("howdy", "partner"), NewElement("hey", "pal")),
				NewVector(NewElement("howdy", "friend"), NewElement("hey", "guy")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("howdy", "partner"), NewElement("hey", "pal")),
				NewVector(NewElement("howdy", "friend"), NewElement("hey", "guy")),
			}},
		},
		"unordered-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: nil,
		},
		"unordered-elements-and-vertices": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("arch", "arm32"), NewElement("backend", "raft")),
				NewVector(NewElement("arch", "amd64"), NewElement("backend", "raft")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
			}},
			expected: nil,
		},
		"equal": {
			in: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			other: &Matrix{Vectors: []*Vector{
				NewVector(NewElement("backend", "raft"), NewElement("arch", "arm32")),
				NewVector(NewElement("backend", "raft"), NewElement("arch", "amd64")),
			}},
			expected: nil,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.expected, test.in.SymmetricDifferenceUnordered(test.other))
		})
	}
}
