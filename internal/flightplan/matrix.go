// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/zclconf/go-cty/cty"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// An Element represents a single element of a matrix vector.
type Element struct {
	Key             string
	Val             string
	formattedString string // cached version of the element as a string
}

// Vector is an ordered collection of matrix elements.
type Vector struct {
	// elements list of elements
	elements []Element
}

// Matrix is an ordered list of vectors. Vectors can be of any length. A matrix can be irregular if
// different length vectors are used.
type Matrix struct {
	Vectors []*Vector
}

// Exclude is an exclusion filter that can be passed to the matrix. It includes the exclusion mode
// to use and a matching vector.
type Exclude struct {
	Mode   pb.Matrix_Exclude_Mode
	Vector *Vector
}

// NewElement takes an element key and value and returns a new Element.
func NewElement(key string, val string) Element {
	return Element{Key: key, Val: val}
}

// NewExclude takes an ExcludeMode and Vector, validates the ExcludeMode and returns a pointer to a
// new instance of Exclude and any errors encountered.
func NewExclude(mode pb.Matrix_Exclude_Mode, vec *Vector) (*Exclude, error) {
	ex := &Exclude{Mode: mode, Vector: vec}

	switch mode {
	case pb.Matrix_Exclude_MODE_EXACTLY,
		pb.Matrix_Exclude_MODE_EQUAL_UNORDERED,
		pb.Matrix_Exclude_MODE_CONTAINS:
	case pb.Matrix_Exclude_MODE_UNSPECIFIED:
		return ex, errors.New("exclusion mode was not specified")
	default:
		return ex, fmt.Errorf("unknown exclusion mode: %d", mode)
	}

	return ex, nil
}

// NewMatrix returns a pointer to a new instance of Matrix.
func NewMatrix() *Matrix {
	return &Matrix{}
}

// NewMatrix takes zero-or-more elements and returns a pointer to a new instance of Vector.
func NewVector(elms ...Element) *Vector {
	return &Vector{
		elements: elms,
	}
}

// NewVectorFromProto takes a proto filter vector and returns a new Vector.
func NewVectorFromProto(pbv *pb.Matrix_Vector) *Vector {
	v := NewVector()
	for _, elm := range pbv.GetElements() {
		v.Add(NewElement(elm.GetKey(), elm.GetValue()))
	}

	return v
}

// String returns the element as a string.
func (e Element) String() string {
	// Matrix and vector comparison operations often required the element as
	// a string. We'll cache it to speed up those operations.
	if e.formattedString != "" {
		return e.formattedString
	}

	e.formattedString = fmt.Sprintf("%s:%s", e.Key, e.Val)

	return e.formattedString
}

// Proto returns the element as a proto message.
func (e Element) Proto() *pb.Matrix_Element {
	return &pb.Matrix_Element{Key: e.Key, Value: e.Val}
}

// Equals compares the element with another Element.
func (e Element) Equal(other Element) bool {
	if e.Key != other.Key {
		return false
	}

	if e.Val != other.Val {
		return false
	}

	return true
}

// Match determines if Exclude directive matches the given Vector.
func (ex *Exclude) Match(vec *Vector) bool {
	if ex == nil || vec == nil {
		return false
	}

	switch ex.Mode {
	case pb.Matrix_Exclude_MODE_EXACTLY:
		if vec.Equal(ex.Vector) {
			return true
		}
	case pb.Matrix_Exclude_MODE_EQUAL_UNORDERED:
		if vec.EqualUnordered(ex.Vector) {
			return true
		}
	case pb.Matrix_Exclude_MODE_CONTAINS:
		if vec.ContainsUnordered(ex.Vector) {
			return true
		}
	case pb.Matrix_Exclude_MODE_UNSPECIFIED:
		return false
	default:
		return false
	}

	return false
}

// Proto returns the exclude as a proto message.
func (ex *Exclude) Proto() *pb.Matrix_Exclude {
	if ex == nil {
		return nil
	}

	return &pb.Matrix_Exclude{
		Vector: ex.Vector.Proto(),
		Mode:   ex.Mode,
	}
}

// FromProto unmarshals a proto Matrix_Exclude into itself.
func (ex *Exclude) FromProto(pfe *pb.Matrix_Exclude) {
	if ex == nil || pfe == nil {
		return
	}

	ex.Vector = NewVectorFromProto(pfe.GetVector())
	ex.Mode = pfe.GetMode()
}

// Add adds an Element to the Vector.
func (v *Vector) Add(e Element) {
	if v == nil {
		return
	}

	if v.elements == nil {
		v.elements = []Element{e}
		return
	}

	v.elements = append(v.elements, e)
}

// Copy creates a new Copy of the Vector.
func (v *Vector) Copy() *Vector {
	if v == nil {
		return nil
	}

	vecC := NewVector()

	if len(v.elements) == 0 {
		return vecC
	}

	vecC.elements = make([]Element, len(v.elements))
	copy(vecC.elements, v.elements)

	return vecC
}

// ContainsUnordered takes a Vector and determines whether all Elements in the given Vector are
// represented in the Vector.
func (v *Vector) ContainsUnordered(other *Vector) bool {
	if v == nil && other == nil {
		return true
	}

	if v == nil {
		return false
	}

	if other == nil {
		return true
	}

	if len(v.elements) < 1 || len(other.elements) < 1 {
		return false
	}

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

// CtyVal returns the vector as a cty.Value. This is lossy as duplicate keys will be overwritten.
func (v *Vector) CtyVal() cty.Value {
	if v == nil {
		return cty.NilVal
	}

	vals := map[string]cty.Value{}
	for _, vec := range v.elements {
		vals[vec.Key] = cty.StringVal(vec.Val)
	}

	return cty.ObjectVal(vals)
}

// Equal returns true if both Vectors have Equal values and Equal value ordering.
func (v *Vector) Equal(other *Vector) bool {
	if v == nil && other == nil {
		return true
	}

	if other == nil {
		return false
	}

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

// EqualUnordered returns true if both Vectors have the same Elements but might not be ordered
// the same. This is useful for Vectors of pairs that do not enforce ordering.
func (v *Vector) EqualUnordered(other *Vector) bool {
	if v == nil && other == nil {
		return true
	}

	if other == nil {
		return false
	}

	if v.elements == nil && other.elements == nil {
		return true
	}

	if v.elements == nil || other.elements == nil {
		return false
	}

	if len(v.elements) != len(other.elements) {
		return false
	}

	// We couldn't determine equality by elements. Next, create a copy of each and sort them to check
	// by element.
	vCpy := v.Copy()
	oCpy := other.Copy()
	vCpy.Sort()
	oCpy.Sort()

	// Make sure our copies didn't result in unexpected nil values.
	if vCpy == nil || oCpy == nil || vCpy.elements == nil || oCpy.elements == nil {
		return false
	}

	// Now check by elements.
	for i := range vCpy.elements {
		if !vCpy.elements[i].Equal(oCpy.elements[i]) {
			return false
		}
	}

	return true
}

// Elements returns a list of the Vectors Elements.
func (v *Vector) Elements() []Element {
	if v == nil {
		return nil
	}

	return v.elements
}

// Proto returns the Vector as a proto message.
func (v *Vector) Proto() *pb.Matrix_Vector {
	if v == nil {
		return nil
	}

	pbv := &pb.Matrix_Vector{Elements: []*pb.Matrix_Element{}}

	if v.elements == nil {
		return pbv
	}

	for _, elm := range v.elements {
		pbv.Elements = append(pbv.GetElements(), &pb.Matrix_Element{
			Key:   elm.Key,
			Value: elm.Val,
		})
	}

	return pbv
}

// Sort sorts the Vector's Elements.
func (v *Vector) Sort() {
	if v == nil || v.elements == nil || len(v.elements) < 2 {
		return
	}

	slices.SortStableFunc(v.elements, compareElement)
}

// String returns the Vector as a string.
func (v *Vector) String() string {
	if v == nil || len(v.elements) < 1 {
		return ""
	}

	if len(v.elements) == 1 {
		return fmt.Sprintf("[%s]", v.elements[0].String())
	}

	b := strings.Builder{}
	b.WriteString("[")
	for i, elm := range v.elements {
		if i != 0 {
			b.WriteString(" ")
		}
		b.WriteString(elm.String())
	}
	b.WriteString("]")

	return b.String()
}

// AddVector adds a Vector the Matrix.
func (m *Matrix) AddVector(vec *Vector) {
	if m == nil || vec == nil || len(vec.elements) == 0 {
		return
	}

	if m.Vectors == nil {
		m.Vectors = []*Vector{vec}
		return
	}

	m.Vectors = append(m.Vectors, vec)
}

// AddVectorSorted adds a sorted Vector to a sorted Matrix.
func (m *Matrix) AddVectorSorted(vec *Vector) {
	if m == nil || vec == nil || len(vec.elements) == 0 {
		return
	}

	if m.Vectors == nil {
		m.Vectors = []*Vector{vec}
		return
	}

	i, _ := slices.BinarySearchFunc(m.Vectors, vec, compareVector)
	m.Vectors = slices.Insert(m.Vectors, i, vec)
}

// GetVectors is an accessor for the vectors.
func (m *Matrix) GetVectors() []*Vector {
	if m == nil {
		return nil
	}

	return m.Vectors
}

// CartesianProduct returns a pointer to a new Matrix whose Vectors are the
// Cartesian product of combining all possible Vector Elements from the Matrix.
func (m *Matrix) CartesianProduct() *Matrix {
	if m == nil {
		return nil
	}

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
		for i := range vlen {
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

// Compact removes duplicate Vectors from a Matrix.
func (m *Matrix) Compact() {
	if m == nil || len(m.Vectors) < 2 {
		return
	}

	vecs := slices.CompactFunc(m.Vectors, func(a, b *Vector) bool {
		return a.Equal(b)
	})

	m.Vectors = vecs
}

// ContainsVectorUnordered returns whether or not a matrix has a Vector whose unordered values
// contain those of the other Vector.
func (m *Matrix) ContainsVectorUnordered(other *Vector) bool {
	if m == nil || other == nil {
		return false
	}

	for _, v := range m.Vectors {
		if v.ContainsUnordered(other) {
			return true
		}
	}

	return false
}

// Copy creates a new copy of the Matrix.
func (m *Matrix) Copy() *Matrix {
	if m == nil {
		return nil
	}

	nm := NewMatrix()

	for i := range m.Vectors {
		nm.AddVector(m.Vectors[i].Copy())
	}

	return nm
}

// Equal returns true if the Matrix and other Matrix have equal verticies.
func (m *Matrix) Equal(other *Matrix) bool {
	if m == nil && other == nil {
		return true
	}

	if m == nil || other == nil {
		return false
	}

	if len(m.Vectors) != len(other.Vectors) {
		return false
	}

	if m.Vectors == nil || other.Vectors == nil {
		return true
	}

	for i := range m.Vectors {
		if m.Vectors[i] == nil && other.Vectors[i] == nil {
			continue
		}

		if m.Vectors[i] == nil {
			return false
		}

		if !m.Vectors[i].Equal(other.Vectors[i]) {
			return false
		}
	}

	return true
}

// EqualUnordered returns true if the Matrix and other Matrix have equal but unordered verticies.
func (m *Matrix) EqualUnordered(other *Matrix) bool {
	if m == nil && other == nil {
		return true
	}

	if (m != nil && other == nil) || (m == nil && other != nil) {
		return false
	}

	if len(m.GetVectors()) != len(other.GetVectors()) {
		return false
	}

	mSorted := m.Copy()
	mSorted.Sort()
	otherSorted := other.Copy()
	otherSorted.Sort()

	return mSorted.Equal(otherSorted)
}

// Exclude takes exclude vararg exclude directives as instances of Exclude. It returns a new
// Matrix with all Exclude directives having been processed on the parent Matrix.
func (m *Matrix) Exclude(Excludes ...*Exclude) *Matrix {
	if m == nil {
		return nil
	}

	if len(Excludes) < 1 {
		return m.Copy()
	}

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

// Filter takes a CcenarioFilter returns a new filtered Matrix.
func (m *Matrix) Filter(filter *ScenarioFilter) *Matrix {
	if m == nil {
		return nil
	}

	if filter == nil {
		return m
	}

	if filter.SelectAll {
		return m.Copy()
	}

	var nm *Matrix
	if filter.Include != nil && len(filter.Include.elements) > 0 {
		// If we have an include filter we'll generate a new sub-matrix with matching vectors
		in := NewMatrix()
		in.AddVector(filter.Include)
		nm = m.IntersectionContainsUnordered(in)
	} else {
		// If we don't have an include and we're not selecting all we need to start with our
		// entire matrix and then process excludes.
		nm = m.Copy()
	}

	if len(filter.Exclude) > 0 {
		nm = nm.Exclude(filter.Exclude...)
	}

	if filter.IntersectionMatrix != nil && len(filter.IntersectionMatrix.Vectors) > 0 {
		nm = nm.IntersectionContainsUnordered(filter.IntersectionMatrix)
	}

	return nm
}

// FromProto takes a proto representation of a Matrix and unmarshals it into itself.
func (m *Matrix) FromProto(in *pb.Matrix) {
	if m == nil || in == nil {
		return
	}

	if len(in.GetVectors()) < 1 {
		return
	}

	if m.Vectors == nil {
		m.Vectors = []*Vector{}
	}

	for i := range in.GetVectors() {
		m.Vectors = append(m.Vectors, NewVectorFromProto(in.GetVectors()[i]))
	}
}

// IntersectionContainsUnordered takes another Matrix and returns a new Matrix whose Vectors are
// composed of the result of a intersection of both matrices vector elements that are contained and
// unordered. It's important to note that contains does not mean equal, so a vector [1,2,3] contains
// [3,1] but a vector of [3,1] does not contain [1,2,3] because it doesn't have all of the elements.
func (m *Matrix) IntersectionContainsUnordered(other *Matrix) *Matrix {
	if m == nil || other == nil {
		return nil
	}

	if len(m.Vectors) < 1 || len(other.Vectors) < 1 {
		return nil
	}

	nm := NewMatrix()
	for mi := range m.Vectors {
		for oi := range other.Vectors {
			if m.Vectors[mi].ContainsUnordered(other.Vectors[oi]) {
				nm.AddVector(m.Vectors[mi])
			}
		}
	}
	for oi := range other.Vectors {
		for mi := range m.Vectors {
			if other.Vectors[oi].ContainsUnordered(m.Vectors[mi]) {
				nm.AddVector(other.Vectors[oi])
			}
		}
	}

	if len(nm.Vectors) < 1 {
		return nil
	}

	return nm.UniqueValues()
}

// HasVector returns whether or not a Matrix has a Vector that exactly matches the Elements of
// another that is given.
func (m *Matrix) HasVector(other *Vector) bool {
	if m == nil || other == nil {
		return false
	}

	for _, v := range m.Vectors {
		if v.Equal(other) {
			return true
		}
	}

	return false
}

// HasVectorSorted returns whether or not a sorted Matrix has a sorted Vector. It assumes the
// Matrix and Vector have both already been sorted.
func (m *Matrix) HasVectorSorted(other *Vector) bool {
	if m == nil || other == nil {
		return false
	}

	_, has := slices.BinarySearchFunc(m.Vectors, other, compareVector)

	return has
}

// HasVectorUnordered returns whether or not a Matrix has a Vector whose unordered values match
// exactly with another that is given.
func (m *Matrix) HasVectorUnordered(other *Vector) bool {
	if m == nil || other == nil {
		return false
	}

	for _, v := range m.Vectors {
		if v.EqualUnordered(other) {
			return true
		}
	}

	return false
}

// Proto returns the Matrix as a proto message. If a Matrix is created with a ScenarioFilter that
// has Includes and Excludes a round trip is lossy and will only retain the Vectors.
func (m *Matrix) Proto() *pb.Matrix {
	if m == nil {
		return nil
	}

	if len(m.Vectors) < 1 {
		return nil
	}

	pbm := &pb.Matrix{
		Vectors: []*pb.Matrix_Vector{},
	}
	for i := range m.Vectors {
		pbm.Vectors = append(pbm.GetVectors(), m.Vectors[i].Proto())
	}

	return pbm
}

// SortVectorElements sorts all the elements of each Vector.
func (m *Matrix) SortVectorElements() {
	if m == nil {
		return
	}

	for i := range m.Vectors {
		m.Vectors[i].Sort()
	}
}

// Sort sorts by all Vectors and Elements included.
func (m *Matrix) Sort() {
	if m == nil || len(m.Vectors) < 1 {
		return
	}

	m.SortVectorElements()
	slices.SortStableFunc(m.Vectors, compareVector)
}

// String returns the Matrix Vectors as a string.
func (m *Matrix) String() string {
	if m == nil || len(m.Vectors) < 1 {
		return ""
	}

	b := strings.Builder{}
	for i := range m.Vectors {
		if i != 0 {
			b.WriteString("\n")
		}
		b.WriteString(m.Vectors[i].String())
	}

	return b.String()
}

// SymmetricDifferenceUnordered returns a new Matrix that includes the symmetric difference between
// two matrices of unordered vertices.
func (m *Matrix) SymmetricDifferenceUnordered(other *Matrix) *Matrix {
	if m == nil && other == nil {
		return nil
	}

	if m == nil && other != nil {
		return other.Copy()
	}

	if m != nil && other == nil {
		return m.Copy()
	}

	nm := NewMatrix()
	for i := range other.Vectors {
		if !m.ContainsVectorUnordered(other.Vectors[i]) {
			nm.AddVector(other.Vectors[i])
		}
	}

	for i := range m.Vectors {
		if !other.ContainsVectorUnordered(m.Vectors[i]) {
			nm.AddVector(m.Vectors[i])
		}
	}

	if len(nm.Vectors) < 1 {
		return nil
	}

	return nm
}

// Unique returns a new Matrix with all unique Vectors.
func (m *Matrix) Unique() *Matrix {
	if m == nil {
		return nil
	}

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
	if m == nil {
		return nil
	}

	if len(m.Vectors) < 2 {
		return m.Copy()
	}

	nmUnsorted := NewMatrix()
	nmSorted := NewMatrix()
	for i := range m.Vectors {
		v := m.Vectors[i].Copy()
		v.Sort()
		if !nmSorted.HasVectorSorted(v) {
			nmSorted.AddVectorSorted(v)
			nmUnsorted.AddVector(m.Vectors[i])
		}
	}

	return nmUnsorted
}

// compareElement takes two Elements and does a sort commparison.
func compareElement(a, b Element) int {
	if iv := cmp.Compare(a.Key, b.Key); iv != 0 {
		return iv
	}

	return cmp.Compare(a.Val, b.Val)
}

// compareVector takes two Vectors and does a sort comparison.
func compareVector(a, b *Vector) int {
	// Compare by existence
	if a == nil && b == nil {
		return 0
	}

	if a != nil && b == nil {
		return 1
	}

	if a == nil && b != nil {
		return -1
	}

	// Compare by number of elements
	aElms := a.Elements()
	bElms := b.Elements()

	if i := cmp.Compare(len(aElms), len(bElms)); i != 0 {
		return i
	}

	// Compare by element existence
	if aElms == nil && bElms == nil {
		return 0
	}

	if aElms != nil && bElms == nil {
		return 1
	}

	if aElms == nil && bElms != nil {
		return -1
	}

	// Compare by element values if we have elements. We do this existence check again to please nilaway
	if aElms != nil && bElms != nil {
		for i, aElm := range aElms {
			bElm := bElms[i]

			if iv := compareElement(aElm, bElm); iv != 0 {
				return iv
			}
		}
	}

	// We have equal vectors
	return 0
}
