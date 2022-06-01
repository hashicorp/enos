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
		in    Vector
		other Vector
		equal bool
	}{
		"equal": {
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			true,
		},
		"ordered but unequal Elements": {
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			false,
		},
		"equal Values": {
			Vector{Element{"backend", "consul"}, Element{"backend", "raft"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
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
		in    Vector
		other Vector
		match bool
	}{
		"exact": {
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			true,
		},
		"unordered unequal len match": {
			Vector{Element{"backend", "consul"}, Element{"backend", "raft"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
			false,
		},
		"unordered exact": {
			Vector{Element{"backend", "consul"}, Element{"backend", "raft"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			true,
		},
		"equal len no match": {
			Vector{Element{"backend", "myssql"}, Element{"backend", "raft"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			false,
		},
		"unequal len no match": {
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
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
		in    Vector
		other Vector
		equal bool
	}{
		"equal": {
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			true,
		},
		"ordered but unequal Elements": {
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
			false,
		},
		"equal Values": {
			Vector{Element{"backend", "consul"}, Element{"backend", "raft"}},
			Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
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
			&Matrix{Vectors: []Vector{}},
			&Matrix{Vectors: nil},
		},
		"regular": {
			&Matrix{
				Vectors: []Vector{
					{Element{"backend", "raft"}, Element{"backend", "consul"}},
					{Element{"arch", "arm64"}, Element{"arch", "amd64"}},
					{Element{"distro", "ubuntu"}, Element{"distro", "rhel"}},
				},
			},
			&Matrix{Vectors: []Vector{
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "ubuntu"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "rhel"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "ubuntu"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "rhel"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "ubuntu"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "rhel"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "ubuntu"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "rhel"}},
			}},
		},
		"irregular": {
			&Matrix{
				Vectors: []Vector{
					{Element{"backend", "raft"}, Element{"backend", "consul"}},
					{Element{"arch", "arm64"}, Element{"arch", "amd64"}, Element{"arch", "ppc64"}},
					{Element{"distro", "ubuntu"}},
					{Element{"test", "fresh-install"}, Element{"test", "upgrade"}, Element{"test", "security"}},
				},
			},
			&Matrix{Vectors: []Vector{
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "fresh-install"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "upgrade"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "security"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "fresh-install"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "upgrade"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "security"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "ppc64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "fresh-install"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "ppc64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "upgrade"}},
				{Element{Key: "backend", Val: "raft"}, Element{Key: "arch", Val: "ppc64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "security"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "fresh-install"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "upgrade"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "arm64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "security"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "fresh-install"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "upgrade"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "amd64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "security"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "ppc64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "fresh-install"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "ppc64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "upgrade"}},
				{Element{Key: "backend", Val: "consul"}, Element{Key: "arch", Val: "ppc64"}, Element{Key: "distro", Val: "ubuntu"}, Element{Key: "test", Val: "security"}},
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
	m.AddVector(Vector{})
	m.AddVector(Vector{})

	require.Equal(t, &Matrix{}, m.CartesianProduct())
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
	for desc, test := range map[string]struct {
		in       *Matrix
		Excludes []*Exclude
		expected *Matrix
	}{
		"exact": {
			&Matrix{Vectors: []Vector{
				{Element{"backend", "raft"}, Element{"backend", "consul"}},
				{Element{"backend", "raft"}, Element{"backend", "consul"}},
				{Element{"backend", "consul"}, Element{"backend", "raft"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
			}},
			[]*Exclude{
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_EXACTLY,
					Vector: Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
				},
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_EXACTLY,
					Vector: Vector{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
				},
			},
			&Matrix{Vectors: []Vector{
				{Element{"backend", "consul"}, Element{"backend", "raft"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}},
			}},
		},
		"equal values": {
			&Matrix{Vectors: []Vector{
				{Element{"backend", "raft"}, Element{"backend", "consul"}},
				{Element{"backend", "raft"}, Element{"backend", "consul"}},
				{Element{"backend", "consul"}, Element{"backend", "raft"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
			}},
			[]*Exclude{
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_EQUAL_UNORDERED,
					Vector: Vector{Element{"backend", "raft"}, Element{"backend", "consul"}},
				},
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_EQUAL_UNORDERED,
					Vector: Vector{Element{"arch", "arm64"}, Element{"arch", "amd64"}},
				},
			},
			&Matrix{Vectors: []Vector{
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
			}},
		},
		"match": {
			&Matrix{Vectors: []Vector{
				{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
				{Element{"backend", "consul"}, Element{"backend", "raft"}, Element{"backend", "mysql"}},
				{Element{"backend", "raft"}, Element{"backend", "mysql"}, Element{"backend", "mssql"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "arm32"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
			}},
			[]*Exclude{
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_CONTAINS,
					Vector: Vector{Element{"backend", "mysql"}},
				},
				{
					Mode:   pb.Scenario_Filter_Exclude_MODE_CONTAINS,
					Vector: Vector{Element{"arch", "arm64"}, Element{"arch", "arm32"}},
				},
			},
			&Matrix{Vectors: []Vector{
				{Element{"backend", "raft"}, Element{"backend", "consul"}, Element{"backend", "mssql"}},
				{Element{"arch", "amd64"}, Element{"arch", "arm64"}, Element{"arch", "ppc64"}},
			}},
		},
	} {
		t.Run(desc, func(t *testing.T) {
			require.Equal(t, test.expected.Vectors, test.in.Exclude(test.Excludes...).Vectors)
		})
	}
}
