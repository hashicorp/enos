package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// Test_Decode_Scenario_Matrix tests decoding of a matrix in scenarios
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
				Scenarios: []*Scenario{
					{
						Name:         "nighttime",
						Variants:     &Vector{unordered: []Element{NewElement("cathat", "conrad"), NewElement("onefish", "twofish")}},
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
						Variants:     &Vector{unordered: []Element{NewElement("cathat", "sally"), NewElement("onefish", "twofish")}},
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
						Variants:     &Vector{unordered: []Element{NewElement("cathat", "thing2"), NewElement("onefish", "bluefish")}},
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
						Variants:     &Vector{unordered: []Element{NewElement("cathat", "thing2"), NewElement("onefish", "redfish")}},
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
	} {
		t.Run(desc, func(t *testing.T) {
			fp, err := testDecodeHCL(t, []byte(test.hcl))
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
	for desc, test := range map[string]struct {
		in    *Vector
		other *Vector
		equal bool
	}{
		"equal": {
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			true,
		},
		"ordered but unequal Elements": {
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			false,
		},
		"equal Values": {
			&Vector{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			false,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			require.Equal(t, test.equal, test.in.Equal(test.other))
		})
	}
}

func Test_Matrix_Vector_ContainsUnordered(t *testing.T) {
	for desc, test := range map[string]struct {
		in    *Vector
		other *Vector
		match bool
	}{
		"exact": {
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			true,
		},
		"unordered unequal len match": {
			&Vector{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")}},
			false,
		},
		"unordered exact": {
			&Vector{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			true,
		},
		"equal len no match": {
			&Vector{unordered: []Element{NewElement("backend", "myssql"), NewElement("backend", "raft")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			false,
		},
		"unequal len no match": {
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")}},
			false,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			require.Equal(t, test.match, test.in.ContainsUnordered(test.other))
		})
	}
}

func Test_Matrix_Vector_EqualUnordered(t *testing.T) {
	for desc, test := range map[string]struct {
		in    *Vector
		other *Vector
		equal bool
	}{
		"equal": {
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			true,
		},
		"ordered but unequal Elements": {
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			false,
		},
		"equal Values": {
			&Vector{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}},
			&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
			true,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			require.Equal(t, test.equal, test.in.EqualUnordered(test.other))
		})
	}
}

func Test_Matrix_CartesianProduct(t *testing.T) {
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
				{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
				{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "amd64")}},
				{unordered: []Element{NewElement("distro", "ubuntu"), NewElement("distro", "rhel")}},
			}},
			&Matrix{Vectors: []*Vector{
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("distro", "rhel")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "amd64"), NewElement("distro", "rhel")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "arm64"), NewElement("distro", "rhel")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "amd64"), NewElement("distro", "rhel")}},
			}},
		},
		"irregular": {
			&Matrix{Vectors: []*Vector{
				{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
				{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "amd64"), NewElement("arch", "ppc64")}},
				{unordered: []Element{NewElement("distro", "ubuntu")}},
				{unordered: []Element{NewElement("test", "fresh-install"), NewElement("test", "upgrade"), NewElement("test", "security")}},
			}},
			&Matrix{Vectors: []*Vector{
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "security")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "security")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "security")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "arm64"), NewElement("distro", "ubuntu"), NewElement("test", "security")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "amd64"), NewElement("distro", "ubuntu"), NewElement("test", "security")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "fresh-install")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "upgrade")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("arch", "ppc64"), NewElement("distro", "ubuntu"), NewElement("test", "security")}},
			}},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			require.Equal(t, test.expected.Vectors, test.in.CartesianProduct().Vectors)
		})
	}
}

func Test_Matrix_CartesianProduct_empty_vector(t *testing.T) {
	m := NewMatrix()
	m.AddVector(NewVector())
	m.AddVector(NewVector())

	require.Equal(t, &Matrix{}, m.CartesianProduct())
}

func Test_Matrix_UniqueValues(t *testing.T) {
	m1 := NewMatrix()
	m1.AddVector(&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "amd64")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "amd64"), NewElement("arch", "ppc64")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64")}})

	m2 := NewMatrix()
	m2.AddVector(&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")}})
	m2.AddVector(&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}})
	m2.AddVector(&Vector{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "amd64")}})
	m2.AddVector(&Vector{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "amd64"), NewElement("arch", "ppc64")}})

	uniq := m1.UniqueValues()
	require.Len(t, uniq.Vectors, len(m2.Vectors))

	for i := range m2.Vectors {
		require.EqualValues(t, m2.Vectors[i].unordered, uniq.Vectors[i].unordered)
	}
}

func Test_Matrix_Unique(t *testing.T) {
	m1 := NewMatrix()
	m1.AddVector(&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft"), NewElement("backend", "myssql")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "amd64")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "amd64")}})
	m1.AddVector(&Vector{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64")}})

	m2 := NewMatrix()
	m2.AddVector(&Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}})
	m2.AddVector(&Vector{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft"), NewElement("backend", "myssql")}})
	m2.AddVector(&Vector{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}})
	m2.AddVector(&Vector{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "amd64")}})
	m2.AddVector(&Vector{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64")}})

	require.Equal(t, m2, m1.Unique())
}

func Test_Matrix_Exclude(t *testing.T) {
	for desc, test := range map[string]struct {
		in       *Matrix
		Excludes []*Exclude
		expected *Matrix
	}{
		"exact": {
			&Matrix{Vectors: []*Vector{
				{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}},
				{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64")}},
				{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")}},
			}},
			[]*Exclude{
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_EXACTLY,
					Vector: &Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
				},
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_EXACTLY,
					Vector: &Vector{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")}},
				},
			},
			&Matrix{Vectors: []*Vector{
				{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}},
				{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64")}},
			}},
		},
		"equal values": {
			&Matrix{Vectors: []*Vector{
				{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft")}},
				{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64")}},
				{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")}},
			}},
			[]*Exclude{
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_EQUAL_UNORDERED,
					Vector: &Vector{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul")}},
				},
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_EQUAL_UNORDERED,
					Vector: &Vector{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "amd64")}},
				},
			},
			&Matrix{Vectors: []*Vector{
				{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")}},
			}},
		},
		"match": {
			&Matrix{Vectors: []*Vector{
				{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")}},
				{unordered: []Element{NewElement("backend", "consul"), NewElement("backend", "raft"), NewElement("backend", "mysql")}},
				{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "mysql"), NewElement("backend", "mssql")}},
				{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "arm32")}},
				{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")}},
			}},
			[]*Exclude{
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_CONTAINS,
					Vector: &Vector{unordered: []Element{NewElement("backend", "mysql")}},
				},
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_CONTAINS,
					Vector: &Vector{unordered: []Element{NewElement("arch", "arm64"), NewElement("arch", "arm32")}},
				},
			},
			&Matrix{Vectors: []*Vector{
				{unordered: []Element{NewElement("backend", "raft"), NewElement("backend", "consul"), NewElement("backend", "mssql")}},
				{unordered: []Element{NewElement("arch", "amd64"), NewElement("arch", "arm64"), NewElement("arch", "ppc64")}},
			}},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			require.Equal(t, test.expected.Vectors, test.in.Exclude(test.Excludes...).Vectors)
		})
	}
}
