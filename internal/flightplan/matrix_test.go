package flightplan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test_Decode_Scenario_Matrix tests decoding of a matrix in scenarios
func Test_Decode_Scenario_Matrix(t *testing.T) {
	t.Parallel()

	modulePath, err := filepath.Abs("./tests/simple_module")
	require.NoError(t, err)

	for _, test := range []struct {
		desc     string
		hcl      string
		expected *FlightPlan
		fail     bool
	}{
		{
			desc: "matrix with label",
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
		{
			desc: "more than one matrix block",
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
		{
			desc: "invalid matrix value",
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
		{
			desc: "invalid block",
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
		{
			desc: "invalid value include",
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
		{
			desc: "invalid block include",
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
		{
			desc: "invalid value exclude",
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

		{
			desc: "invalid block exclude",
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
		{
			desc: "valid matrix",
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
						Variants:     Vector{NewElement("cathat", "conrad"), NewElement("onefish", "twofish")},
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
						Variants:     Vector{NewElement("cathat", "sally"), NewElement("onefish", "twofish")},
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
						Variants:     Vector{NewElement("cathat", "thing2"), NewElement("onefish", "bluefish")},
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
						Variants:     Vector{NewElement("cathat", "thing2"), NewElement("onefish", "redfish")},
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
		t.Run(test.desc, func(t *testing.T) {
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
	for _, test := range []struct {
		desc  string
		in    Vector
		other Vector
		equal bool
	}{
		{
			"equal",
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			true,
		},
		{
			"ordered but unequal Elements",
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			false,
		},
		{
			"equal Values",
			Vector{Element{"backend", "consul"}, Element{"backend", "raft"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			false,
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			require.Equal(t, test.equal, test.in.Equal(test.other))
		})
	}
}

func Test_Matrix_Vector_ContainsValues(t *testing.T) {
	for _, test := range []struct {
		desc  string
		in    Vector
		other Vector
		match bool
	}{
		{
			"exact",
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			true,
		},
		{
			"unordered unequal len match",
			Vector{Element{"backend", "consul"}, Element{"backend", "raft"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
			false,
		},
		{
			"unordered exact",
			Vector{Element{"backend", "consul"}, Element{"backend", "raft"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			true,
		},
		{
			"equal len no match",
			Vector{Element{"backend", "myssql"}, Element{"backend", "raft"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			false,
		},
		{
			"unequal len no match",
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
			false,
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			require.Equal(t, test.match, test.in.ContainsValues(test.other))
		})
	}
}

func Test_Matrix_Vector_EqualValues(t *testing.T) {
	for _, test := range []struct {
		desc  string
		in    Vector
		other Vector
		equal bool
	}{
		{
			"equal",
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			true,
		},
		{
			"ordered but unequal Elements",
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			false,
		},
		{
			"equal Values",
			Vector{Element{"backend", "consul"}, Element{"backend", "raft"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			true,
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			require.Equal(t, test.equal, test.in.EqualValues(test.other))
		})
	}
}

func Test_Matrix_CombinedVectors(t *testing.T) {
	m := NewMatrix()
	m.AddVector(Vector{Element{"backend", "raft"}, Element{"backend", "consul"}})
	m.AddVector(Vector{Element{"arch", "arm64"}, Element{"arch", "amd64"}})
	m.AddVector(Vector{Element{"distro", "ubuntu"}, Element{"arch", "rhel"}})

	require.Equal(t, []Vector{
		{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "ubuntu"}},
		{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "arch", Val: "rhel"}},
		{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "ubuntu"}},
		{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "arch", Val: "rhel"}},
		{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "ubuntu"}},
		{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "arch", Val: "rhel"}},
		{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "ubuntu"}},
		{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "arch", Val: "rhel"}},
	},
		m.CombinedVectors().Vectors,
	)
}

func Test_Matrix_CombinedVectors_empty_vector(t *testing.T) {
	m := NewMatrix()
	m.AddVector(Vector{})
	m.AddVector(Vector{})

	require.Equal(t, &Matrix{}, m.CombinedVectors())
}

func Test_Matrix_UniqueValues(t *testing.T) {
	m1 := NewMatrix()
	m1.AddVector(Vector{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}})
	m1.AddVector(Vector{Element{"backend", "raft"}, Element{"backend", "consul"}})
	m1.AddVector(Vector{Element{"backend", "consul"}, Element{"backend", "raft"}})
	m1.AddVector(Vector{Element{"arch", "arm64"}, Element{"arch", "amd64"}})
	m1.AddVector(Vector{Element{"arch", "arm64"}, Element{"arch", "amd64"}, Element{"arch", "ppc64"}})
	m1.AddVector(Vector{Element{"arch", "amd64"}, Element{"arch", "arm64"}})

	m2 := NewMatrix()
	m2.AddVector(Vector{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}})
	m2.AddVector(Vector{Element{"backend", "raft"}, Element{"backend", "consul"}})
	m2.AddVector(Vector{Element{"arch", "arm64"}, Element{"arch", "amd64"}})
	m2.AddVector(Vector{Element{"arch", "arm64"}, Element{"arch", "amd64"}, Element{"arch", "ppc64"}})

	require.EqualValues(t, m2.Vectors, m1.UniqueValues().Vectors)
}

func Test_Matrix_Unique(t *testing.T) {
	m1 := NewMatrix()
	m1.AddVector(Vector{Element{"backend", "raft"}, Element{"backend", "consul"}})
	m1.AddVector(Vector{Element{"backend", "raft"}, Element{"backend", "consul"}})
	m1.AddVector(Vector{Element{"backend", "consul"}, Element{"backend", "raft"}, Element{"backend", "myssql"}})
	m1.AddVector(Vector{Element{"backend", "consul"}, Element{"backend", "raft"}})
	m1.AddVector(Vector{Element{"backend", "consul"}, Element{"backend", "raft"}})
	m1.AddVector(Vector{Element{"arch", "arm64"}, Element{"arch", "amd64"}})
	m1.AddVector(Vector{Element{"arch", "arm64"}, Element{"arch", "amd64"}})
	m1.AddVector(Vector{Element{"arch", "amd64"}, Element{"arch", "arm64"}})

	m2 := NewMatrix()
	m2.AddVector(Vector{Element{"backend", "raft"}, Element{"backend", "consul"}})
	m2.AddVector(Vector{Element{"backend", "consul"}, Element{"backend", "raft"}, Element{"backend", "myssql"}})
	m2.AddVector(Vector{Element{"backend", "consul"}, Element{"backend", "raft"}})
	m2.AddVector(Vector{Element{"arch", "arm64"}, Element{"arch", "amd64"}})
	m2.AddVector(Vector{Element{"arch", "amd64"}, Element{"arch", "arm64"}})

	require.Equal(t, m2, m1.Unique())
}

func Test_Matrix_Exclude(t *testing.T) {
	for _, test := range []struct {
		desc     string
		in       *Matrix
		Excludes []*Exclude
		expected *Matrix
	}{
		{
			"exact",
			&Matrix{Vectors: []Vector{
				{Element{"backend", "raft"}, Element{"backend", "consul"}},
				{Element{"backend", "raft"}, Element{"backend", "consul"}},
				{Element{"backend", "consul"}, Element{"backend", "raft"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
			}},
			[]*Exclude{
				{
					Mode:   ExcludeExactly,
					Vector: Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
				},
				{
					Mode:   ExcludeExactly,
					Vector: Vector{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
				},
			},
			&Matrix{Vectors: []Vector{
				{Element{"backend", "consul"}, Element{"backend", "raft"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}},
			}},
		},
		{
			"equal Values",
			&Matrix{Vectors: []Vector{
				{Element{"backend", "raft"}, Element{"backend", "consul"}},
				{Element{"backend", "raft"}, Element{"backend", "consul"}},
				{Element{"backend", "consul"}, Element{"backend", "raft"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
			}},
			[]*Exclude{
				{
					Mode:   ExcludeEqualValues,
					Vector: Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
				},
				{
					Mode:   ExcludeEqualValues,
					Vector: Vector{Element{"arch", "arm64"}, Element{"arch", "amd64"}},
				},
			},
			&Matrix{Vectors: []Vector{
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
			}},
		},
		{
			"match",
			&Matrix{Vectors: []Vector{
				{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
				{Element{"backend", "consul"}, Element{"backend", "raft"}, Element{"backend", "mysql"}},
				{Element{"backend", "raft"}, Element{"backend", "mysql"}, Element{"backend", "mssql"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "arm32"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
			}},
			[]*Exclude{
				{
					Mode:   ExcludeMatch,
					Vector: Vector{Element{"backend", "mysql"}},
				},
				{
					Mode:   ExcludeMatch,
					Vector: Vector{Element{"arch", "arm64"}, Element{"arch", "arm32"}},
				},
			},
			&Matrix{Vectors: []Vector{
				{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
			}},
		},
	} {
		require.Equal(t, test.expected.Vectors, test.in.Exclude(test.Excludes...).Vectors)
	}
}
