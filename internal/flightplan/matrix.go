package flightplan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"
)

// An Element is an Element of a Matrix Vector
type Element struct {
	Key string
	Val string
}

// Vector is a collection of Matrix Elements.
type Vector []Element

// A Matrix contains an slice of Vectors. The collection of Vectors can be
// used to form a regular or irregular Matrix.
type Matrix struct {
	Vectors []Vector
}

// ExcludeMode determines how we're going match Vectors to Exclude
type ExcludeMode int

const (
	// ExcludeExactly will match a vector that has the exact ordered elements
	ExcludeExactly ExcludeMode = iota + 1
	// ExcludeEqualValues will match a vector that has the exact elements but may
	// be unordered.
	ExcludeEqualValues
	// ExcludeMatch will match any vector that has at least the given vector
	// elements in any order.
	ExcludeMatch
)

// An Exclude is a filter to removing Elements from the Matrix's Vector combined
type Exclude struct {
	Mode   ExcludeMode
	Vector Vector
}

// NewElement takes an element key and value and returns a new Element
func NewElement(key string, val string) Element {
	return Element{Key: key, Val: val}
}

// NewMatrix returns a pointer to a new instance of Matrix
func NewMatrix() *Matrix {
	return &Matrix{}
}

// NewExclude takes an ExcludeMode and Vector, validates the ExcludeMode and return
// a pointer to a new instance of Exclude and any errors encountered.
func NewExclude(mode ExcludeMode, vec Vector) (*Exclude, error) {
	ex := &Exclude{Mode: mode, Vector: vec}

	switch mode {
	case ExcludeExactly, ExcludeEqualValues, ExcludeMatch:
	default:
		return ex, fmt.Errorf("unknown exclusion mode: %d", mode)
	}

	return ex, nil
}

// String returns the element as a string
func (e Element) String() string {
	return fmt.Sprintf("%s:%s", e.Key, e.Val)
}

// String returns the vector as a string
func (v Vector) String() string {
	elmStrings := []string{}
	for _, elm := range v {
		elmStrings = append(elmStrings, elm.String())
	}

	return fmt.Sprintf("[%s]", strings.Join(elmStrings, " "))
}

// Equal returns true if both Vectors have Equal values and Equal value ordering
func (v Vector) Equal(other Vector) bool {
	if len(v) != len(other) {
		return false
	}

	for i, ve := range v {
		if ve != other[i] {
			return false
		}
	}

	return true
}

// EqualValues returns true if both Vectors have the same Elements but might
// not be ordered the same. This is useful for Vectors of pairs that do not
// enforce ordering.
func (v Vector) EqualValues(other Vector) bool {
	if len(v) != len(other) {
		return false
	}

	// Go slice values are header references to backing arrays. As such, we need
	// to make copies of each Vector because sorting them without copying them
	// to new memory will modify the backing arrays.
	vC := make(Vector, len(v))
	copy(vC, v)

	otherC := make(Vector, len(other))
	copy(otherC, other)

	sort.Slice(vC, func(i, j int) bool {
		return vC[i].String() < vC[j].String()
	})
	sort.Slice(otherC, func(i, j int) bool {
		return otherC[i].String() < otherC[j].String()
	})

	return vC.Equal(otherC)
}

// ContainsValues returns a boolean which represent if vector contains the values
// of another vector.
func (v Vector) ContainsValues(other Vector) bool {
	for _, otherElm := range other {
		match := false
		for _, elm := range v {
			if otherElm.Key == elm.Key && otherElm.Val == elm.Val {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	return true
}

// CtyVal returns the vector as a cty.Value. Note that this is lossy as duplicate
// keys will be overwritten.
func (v Vector) CtyVal() cty.Value {
	vals := map[string]cty.Value{}
	for _, vec := range v {
		vals[vec.Key] = cty.StringVal(vec.Val)
	}

	return cty.ObjectVal(vals)
}

// AddVector adds a vector the the matrix.
func (m *Matrix) AddVector(vec Vector) {
	if m.Vectors == nil {
		m.Vectors = []Vector{}
	}

	if len(vec) == 0 {
		return
	}

	// Always make a copy of each Vector so we don't accidentally refer to the
	// same backing array when adding a Vector from one Matrix to another.
	vecC := make(Vector, len(vec))
	copy(vecC, vec)
	m.Vectors = append(m.Vectors, vecC)
}

// ExcludeVector determines if Exclude directive matches the vector
func (ex *Exclude) ExcludeVector(vec Vector) bool {
	switch ex.Mode {
	case ExcludeExactly:
		if vec.Equal(ex.Vector) {
			return true
		}
	case ExcludeEqualValues:
		if vec.EqualValues(ex.Vector) {
			return true
		}
	case ExcludeMatch:
		if vec.ContainsValues(ex.Vector) {
			return true
		}
	default:
	}

	return false
}

// Exclude takes exclude vararg exclude directives as instances of Exclude. It
// returns a new matrix with all exclude directives having been processed on
// on the parent matrix.
func (m *Matrix) Exclude(Excludes ...*Exclude) *Matrix {
	nm := NewMatrix()

	for _, vec := range m.Vectors {
		skip := false
		for _, ex := range Excludes {
			if ex.ExcludeVector(vec) {
				skip = true
				break
			}
		}
		if !skip {
			nm.AddVector(vec)
		}
	}

	return nm
}

// CombinedVectors creates a new Matrix of the product of combining all possible
// Vectors unique combinations.
func (m *Matrix) CombinedVectors() *Matrix {
	combined := NewMatrix()
	vlen := len(m.Vectors)
	if vlen == 0 {
		return combined
	}
	// vecIdx is where we'll keep track of which index to use in a given Vector
	// for our combined Vector.
	vecIdx := make([]int, vlen)

	for {
		// Create our Vector by taking the index address of each Element and
		// getting the value out of the corresponding Vector.
		vec := Vector{}
		for i := 0; i < vlen; i++ {
			vec = append(vec, m.Vectors[i][vecIdx[i]])
		}
		combined.Vectors = append(combined.Vectors, vec)

		// Find the Vector we need to increase by searching backwards through
		// all Vectors until we run into a Vector whose Element index cannot be
		// incremented.
		next := vlen - 1
		for {
			if next >= 0 && (vecIdx[next]+1 >= len(m.Vectors[next])) {
				next = next - 1
			} else {
				break
			}
		}

		// We cannot walk back past our current Vector and our current Vector
		// cannot be incremented. We're done.
		if next < 0 {
			break
		}

		// Increment our selected Vectors Element index.
		vecIdx[next]++

		// Reset all Elements in Vectors past our chosen Element to zero.
		for i := next + 1; i < vlen; i++ {
			vecIdx[i] = 0
		}
	}

	return combined
}

// HasVector returns whether or not a matrix has a vector that exactly matches
// the elements of another that is given.
func (m *Matrix) HasVector(other Vector) bool {
	for _, v := range m.Vectors {
		if v.Equal(other) {
			return true
		}
	}

	return false
}

// HasVectorValues returns whether or not a matrix has a vector whose unordered
// values match exactly with another that is given.
func (m *Matrix) HasVectorValues(other Vector) bool {
	for _, v := range m.Vectors {
		if v.EqualValues(other) {
			return true
		}
	}

	return false
}

// Unique returns a new Matrix with all unique Vectors.
func (m *Matrix) Unique() *Matrix {
	nm := NewMatrix()
	for _, v := range m.Vectors {
		if !nm.HasVector(v) {
			nm.AddVector(v)
		}
	}

	return nm
}

// UniqueValues returns a new Matrix with all Vectors that have unique values.
func (m *Matrix) UniqueValues() *Matrix {
	nm := NewMatrix()
	for _, v := range m.Vectors {
		if !nm.HasVectorValues(v) {
			nm.AddVector(v)
		}
	}

	return nm
}
