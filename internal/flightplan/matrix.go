package flightplan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// An Element is an Element of a Matrix Vector
type Element struct {
	Key             string
	Val             string
	formattedString string // cached version of the element as a string
}

// Vector is a collection of Matrix Elements. The Vector maintains orignal
// ordering in the elements array and optionally keeps a cached sorted
// array for comparison operations.
type Vector struct {
	// elements list of elements
	elements []Element
	// an sorted set of elements that we'll populate for some comparisons
	sorted []Element
	// whether or not our vector has been modified and needs to be resorted
	// before some comparison operations.
	dirty bool
}

// A Matrix contains an slice of Vectors. The collection of Vectors can be
// used to form a regular or irregular Matrix.
type Matrix struct {
	Vectors []*Vector
}

// An Exclude is a filter to removing Elements from the Matrix's Vector combined
type Exclude struct {
	Mode   pb.Scenario_Filter_Exclude_Mode
	Vector *Vector
}

// NewElement takes an element key and value and returns a new Element
func NewElement(key string, val string) Element {
	return Element{Key: key, Val: val}
}

// NewMatrix returns a pointer to a new instance of Matrix
func NewMatrix() *Matrix {
	return &Matrix{}
}

// NewExclude takes an ExcludeMode and Vector, validates the ExcludeMode and
// return a pointer to a new instance of Exclude and any errors encountered.
func NewExclude(mode pb.Scenario_Filter_Exclude_Mode, vec *Vector) (*Exclude, error) {
	ex := &Exclude{Mode: mode, Vector: vec}

	switch mode {
	case pb.Scenario_Filter_Exclude_MODE_EXACTLY,
		pb.Scenario_Filter_Exclude_MODE_EQUAL_UNORDERED,
		pb.Scenario_Filter_Exclude_MODE_CONTAINS:
	default:
		return ex, fmt.Errorf("unknown exclusion mode: %d", mode)
	}

	return ex, nil
}

// String returns the element as a string
func (e Element) String() string {
	// Matrix and vector comparison operations often required the element as
	// a string. We'll cache it to speed up those operations.
	if e.formattedString != "" {
		return e.formattedString
	}

	e.formattedString = fmt.Sprintf("%s:%s", e.Key, e.Val)

	return e.formattedString
}

// Proto returns the element as a proto message
func (e Element) Proto() *pb.Scenario_Filter_Element {
	return &pb.Scenario_Filter_Element{Key: e.Key, Value: e.Val}
}

// Equals compares the element with another
func (e Element) Equal(other Element) bool {
	if e.Key != other.Key {
		return false
	}

	if e.Val != other.Val {
		return false
	}

	return true
}

// NewElementFromProto creates a new Element from a proto filter element
func NewElementFromProto(p *pb.Scenario_Filter_Element) Element {
	return NewElement(p.GetKey(), p.GetValue())
}

func NewVector() *Vector {
	return &Vector{}
}

// String returns the vector as a string
func (v *Vector) String() string {
	elmStrings := []string{}
	for _, elm := range v.elements {
		elmStrings = append(elmStrings, elm.String())
	}

	return fmt.Sprintf("[%s]", strings.Join(elmStrings, " "))
}

// Equal returns true if both Vectors have Equal values and Equal value ordering
func (v *Vector) Equal(other *Vector) bool {
	if v.elements == nil && other.elements == nil {
		return true
	}

	if v.elements == nil || other.elements == nil {
		return false
	}

	if len(v.elements) != len(other.elements) {
		return false
	}

	for i := range v.elements {
		if v.elements[i] != other.elements[i] {
			return false
		}
	}

	return true
}

// EqualUnordered returns true if both Vectors have the same Elements but might
// not be ordered the same. This is useful for Vectors of pairs that do not
// enforce ordering.
func (v *Vector) EqualUnordered(other *Vector) bool {
	if v.elements == nil && other.elements == nil {
		return true
	}

	if v.elements == nil || other.elements == nil {
		return false
	}

	if len(v.elements) != len(other.elements) {
		return false
	}

	v.sort()
	other.sort()

	for i := range v.sorted {
		if v.sorted[i] != other.sorted[i] {
			return false
		}
	}

	return true
}

// ContainsUnordered returns a boolean which represent if vector contains the values
// of another vector.
func (v *Vector) ContainsUnordered(other *Vector) bool {
	for oi := range other.elements {
		match := false
		for vi := range v.elements {
			if other.elements[oi].Equal(v.elements[vi]) {
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
func (v *Vector) CtyVal() cty.Value {
	vals := map[string]cty.Value{}
	for _, vec := range v.elements {
		vals[vec.Key] = cty.StringVal(vec.Val)
	}

	return cty.ObjectVal(vals)
}

// Proto returns the vector as a proto message
func (v *Vector) Proto() *pb.Scenario_Filter_Vector {
	pbv := &pb.Scenario_Filter_Vector{Elements: []*pb.Scenario_Filter_Element{}}

	if v == nil || v.elements == nil {
		return pbv
	}

	for _, elm := range v.elements {
		pbv.Elements = append(pbv.Elements, &pb.Scenario_Filter_Element{
			Key:   elm.Key,
			Value: elm.Val,
		})
	}

	return pbv
}

// Add adds an element to the Vector
func (v *Vector) Add(e Element) {
	if v.elements == nil {
		v.elements = []Element{}
	}

	v.elements = append(v.elements, e)

	if v.sorted != nil {
		v.sorted = append(v.sorted, e)
		v.dirty = true
	}
}

// Copy creates a new Copy of the Vector
func (v *Vector) Copy() *Vector {
	vecC := NewVector()

	if v.elements == nil || len(v.elements) == 0 {
		return vecC
	}

	vecC.dirty = v.dirty
	vecC.elements = make([]Element, len(v.elements))
	copy(vecC.elements, v.elements)

	if v.sorted != nil && len(v.sorted) > 0 {
		vecC.sorted = make([]Element, len(v.sorted))
		copy(vecC.sorted, v.sorted)
	}

	return vecC
}

// Elements returns a list of the vectors elements
func (v *Vector) Elements() []Element {
	return v.elements
}

// SortedElements returns a list of vectors elements that have been sorted.
// This can be used for unordered comparisons.
func (v *Vector) SortedElements() []Element {
	v.sort()

	return v.sorted
}

func (v *Vector) sort() {
	if v.elements == nil {
		return
	}

	if v.sorted == nil {
		v.dirty = true
		v.sorted = make([]Element, len(v.elements))
		copy(v.sorted, v.elements)
	}

	if !v.dirty {
		return
	}

	sort.Slice(v.sorted, func(i, j int) bool {
		return v.sorted[i].String() < v.sorted[j].String()
	})

	v.dirty = false
}

// NewVectorFromProto takes a proto filter vector and returns a new Vector.
func NewVectorFromProto(pbv *pb.Scenario_Filter_Vector) *Vector {
	v := NewVector()
	for _, elm := range pbv.GetElements() {
		v.Add(NewElement(elm.GetKey(), elm.GetValue()))
	}
	return v
}

// AddVector adds a vector the matrix.
func (m *Matrix) AddVector(vec *Vector) {
	if vec == nil || len(vec.elements) == 0 {
		return
	}

	if m.Vectors == nil {
		m.Vectors = []*Vector{}
	}

	m.Vectors = append(m.Vectors, vec)
}

// Exclude takes exclude vararg exclude directives as instances of Exclude. It
// returns a new matrix with all exclude directives having been processed on
// on the parent matrix.
func (m *Matrix) Exclude(Excludes ...*Exclude) *Matrix {
	nm := NewMatrix()

	for _, vec := range m.Vectors {
		skip := false
		for _, ex := range Excludes {
			if ex.Match(vec) {
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

// CartesianProduct returns a pointer to a new Matrix whose Vectors are the
// Cartesian product of combining all possible Vector Elements from the Matrix.
func (m *Matrix) CartesianProduct() *Matrix {
	product := NewMatrix()
	vlen := len(m.Vectors)
	if vlen == 0 {
		return product
	}
	// vecIdx is where we'll keep track the Element index for each Vector.
	vecIdx := make([]int, vlen)

	for {
		// Create our next product Vector by reading our Element index address
		// for each Vector in our vector index.
		vec := NewVector()
		for i := 0; i < vlen; i++ {
			vec.Add(m.Vectors[i].elements[vecIdx[i]])
		}
		product.Vectors = append(product.Vectors, vec)

		// Starting from the last Vector in the Matrix, walk backwards until
		// we find a Vector's whose element index can be incremented.
		next := vlen - 1
		for {
			if next >= 0 && (vecIdx[next]+1 >= len(m.Vectors[next].elements)) {
				// We can't increment this Vector, keep walking back
				next = next - 1
			} else {
				// We found a Vector index to increment or we've exhausted our
				// search for a Vector that can be incremented.
				break
			}
		}

		// We walked back past the first Vector. We're done.
		if next < 0 {
			break
		}

		// Increment the Element index for the Vector we walked back to.
		vecIdx[next]++

		// Reset all Element indices in Vectors past our walked to Vector.
		for i := next + 1; i < vlen; i++ {
			vecIdx[i] = 0
		}
	}

	return product
}

// HasVector returns whether or not a matrix has a vector that exactly matches
// the elements of another that is given.
func (m *Matrix) HasVector(other *Vector) bool {
	for _, v := range m.Vectors {
		if v.Equal(other) {
			return true
		}
	}

	return false
}

// HasVectorUnordered returns whether or not a matrix has a vector whose unordered
// values match exactly with another that is given.
func (m *Matrix) HasVectorUnordered(other *Vector) bool {
	for _, v := range m.Vectors {
		if v.EqualUnordered(other) {
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
		if !nm.HasVectorUnordered(v) {
			nm.AddVector(v)
		}
	}

	return nm
}

// Match determines if Exclude directive matches the vector
func (ex *Exclude) Match(vec *Vector) bool {
	switch ex.Mode {
	case pb.Scenario_Filter_Exclude_MODE_EXACTLY:
		if vec.Equal(ex.Vector) {
			return true
		}
	case pb.Scenario_Filter_Exclude_MODE_EQUAL_UNORDERED:
		if vec.EqualUnordered(ex.Vector) {
			return true
		}
	case pb.Scenario_Filter_Exclude_MODE_CONTAINS:
		if vec.ContainsUnordered(ex.Vector) {
			return true
		}
	default:
	}

	return false
}

// Proto returns the exclude as a proto message
func (ex *Exclude) Proto() *pb.Scenario_Filter_Exclude {
	return &pb.Scenario_Filter_Exclude{
		Vector: ex.Vector.Proto(),
		Mode:   ex.Mode,
	}
}

// FromProto unmarshals a proto Scenario_Filter_Exclude into itself
func (ex *Exclude) FromProto(pfe *pb.Scenario_Filter_Exclude) {
	ex.Vector = NewVectorFromProto(pfe.GetVector())
	ex.Mode = pfe.GetMode()
}
