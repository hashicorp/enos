// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package funcs

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/zclconf/go-cty/cty/gocty"
)

// AllTrueFunc constructs a function that returns true if all elements of the list are true. If the
// list is empty, return true.
var AllTrueFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.Bool),
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		result := cty.True
		for it := args[0].ElementIterator(); it.Next(); {
			_, v := it.Element()
			if !v.IsKnown() {
				return cty.UnknownVal(cty.Bool), nil
			}
			if v.IsNull() {
				return cty.False, nil
			}
			result = result.And(v)
			if result.False() {
				return cty.False, nil
			}
		}

		return result, nil
	},
})

// AnyTrueFunc constructs a function that returns true if any element of the list is true. If the
// list is empty, return false.
var AnyTrueFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.Bool),
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		result := cty.False
		var hasUnknown bool
		for it := args[0].ElementIterator(); it.Next(); {
			_, v := it.Element()
			if !v.IsKnown() {
				hasUnknown = true
				continue
			}
			if v.IsNull() {
				continue
			}
			result = result.Or(v)
			if result.True() {
				return cty.True, nil
			}
		}
		if hasUnknown {
			return cty.UnknownVal(cty.Bool), nil
		}

		return result, nil
	},
})

// MatchkeysFunc constructs a function that constructs a new list by taking a subset of elements
// from one list whose indexes match the corresponding indexes of values in another list.
var MatchkeysFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "values",
			Type: cty.List(cty.DynamicPseudoType),
		},
		{
			Name: "keys",
			Type: cty.List(cty.DynamicPseudoType),
		},
		{
			Name: "searchset",
			Type: cty.List(cty.DynamicPseudoType),
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		ty, _ := convert.UnifyUnsafe([]cty.Type{args[1].Type(), args[2].Type()})
		if ty == cty.NilType {
			return cty.NilType, errors.New("keys and searchset must be of the same type")
		}

		// the return type is based on args[0] (values)
		return args[0].Type(), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		if !args[0].IsKnown() {
			return cty.UnknownVal(cty.List(retType.ElementType())), nil
		}

		if args[0].LengthInt() != args[1].LengthInt() {
			return cty.ListValEmpty(retType.ElementType()), errors.New("length of keys and values should be equal")
		}

		output := make([]cty.Value, 0)
		values := args[0]

		// Keys and searchset must be the same type.
		// We can skip error checking here because we've already verified that
		// they can be unified in the Type function
		ty, _ := convert.UnifyUnsafe([]cty.Type{args[1].Type(), args[2].Type()})
		keys, _ := convert.Convert(args[1], ty)
		searchset, _ := convert.Convert(args[2], ty)

		// if searchset is empty, return an empty list.
		if searchset.LengthInt() == 0 {
			return cty.ListValEmpty(retType.ElementType()), nil
		}

		if !values.IsWhollyKnown() || !keys.IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
		}

		i := 0
		for it := keys.ElementIterator(); it.Next(); {
			_, key := it.Element()
			for iter := searchset.ElementIterator(); iter.Next(); {
				_, search := iter.Element()
				eq, err := stdlib.Equal(key, search)
				if err != nil {
					return cty.NilVal, err
				}
				if !eq.IsKnown() {
					return cty.ListValEmpty(retType.ElementType()), nil
				}
				if eq.True() {
					v := values.Index(cty.NumberIntVal(int64(i)))
					output = append(output, v)

					break
				}
			}
			i++
		}

		// if we haven't matched any key, then output is an empty list.
		if len(output) == 0 {
			return cty.ListValEmpty(retType.ElementType()), nil
		}

		return cty.ListVal(output), nil
	},
})

// OneFunc returns either the first element of a one-element list, or null if given a zero-element
// list.
var OneFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.DynamicPseudoType,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		ty := args[0].Type()
		switch {
		case ty.IsListType() || ty.IsSetType():
			return ty.ElementType(), nil
		case ty.IsTupleType():
			etys := ty.TupleElementTypes()
			switch len(etys) {
			case 0:
				// No specific type information, so we'll ultimately return
				// a null value of unknown type.
				return cty.DynamicPseudoType, nil
			case 1:
				return etys[0], nil
			}
		}

		return cty.NilType, function.NewArgErrorf(0, "must be a list, set, or tuple value with either zero or one elements")
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		val := args[0]
		ty := val.Type()

		// Our parameter spec above doesn't set AllowUnknown or AllowNull,
		// so we can assume our top-level collection is both known and non-null
		// in here.

		switch {
		case ty.IsListType() || ty.IsSetType():
			lenVal := val.Length()
			if !lenVal.IsKnown() {
				return cty.UnknownVal(retType), nil
			}
			var l int
			err := gocty.FromCtyValue(lenVal, &l)
			if err != nil {
				// It would be very strange to get here, because that would
				// suggest that the length is either not a number or isn't
				// an integer, which would suggest a bug in cty.
				return cty.NilVal, fmt.Errorf("invalid collection length: %s", err)
			}
			switch l {
			case 0:
				return cty.NullVal(retType), nil
			case 1:
				var ret cty.Value
				// We'll use an iterator here because that works for both lists
				// and sets, whereas indexing directly would only work for lists.
				// Since we've just checked the length, we should only actually
				// run this loop body once.
				for it := val.ElementIterator(); it.Next(); {
					_, ret = it.Element()
				}

				return ret, nil
			}
		case ty.IsTupleType():
			etys := ty.TupleElementTypes()
			switch len(etys) {
			case 0:
				return cty.NullVal(retType), nil
			case 1:
				ret := val.Index(cty.NumberIntVal(0))
				return ret, nil
			}
		}

		return cty.NilVal, function.NewArgErrorf(0, "must be a list, set, or tuple value with either zero or one elements")
	},
})

// SumFunc constructs a function that returns the sum of all numbers provided in a list.
var SumFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.DynamicPseudoType,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		if !args[0].CanIterateElements() {
			return cty.NilVal, function.NewArgErrorf(0, "cannot sum noniterable")
		}

		if args[0].LengthInt() == 0 { // Easy path
			return cty.NilVal, function.NewArgErrorf(0, "cannot sum an empty list")
		}

		arg := args[0].AsValueSlice()
		ty := args[0].Type()

		if !ty.IsListType() && !ty.IsSetType() && !ty.IsTupleType() {
			return cty.NilVal, function.NewArgErrorf(0, "argument must be list, set, or tuple. Received %s", ty.FriendlyName())
		}

		if !args[0].IsWhollyKnown() {
			return cty.UnknownVal(cty.Number), nil
		}

		// big.Float.Add can panic if the input values are opposing infinities,
		// so we must catch that here in order to remain within
		// the cty Function abstraction.
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(big.ErrNaN); ok {
					ret = cty.NilVal
					err = errors.New("can't compute sum of opposing infinities")
				} else {
					// not a panic we recognize
					panic(r)
				}
			}
		}()

		s := arg[0]
		if s.IsNull() {
			return cty.NilVal, function.NewArgErrorf(0, "argument must be list, set, or tuple of number values")
		}
		s, err = convert.Convert(s, cty.Number)
		if err != nil {
			return cty.NilVal, function.NewArgErrorf(0, "argument must be list, set, or tuple of number values")
		}
		for _, v := range arg[1:] {
			if v.IsNull() {
				return cty.NilVal, function.NewArgErrorf(0, "argument must be list, set, or tuple of number values")
			}
			v, err = convert.Convert(v, cty.Number)
			if err != nil {
				return cty.NilVal, function.NewArgErrorf(0, "argument must be list, set, or tuple of number values")
			}
			s = s.Add(v)
		}

		return s, nil
	},
})

// TransposeFunc constructs a function that takes a map of lists of strings and swaps the keys and
// values to produce a new map of lists of strings.
var TransposeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "values",
			Type: cty.Map(cty.List(cty.String)),
		},
	},
	Type: function.StaticReturnType(cty.Map(cty.List(cty.String))),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		inputMap := args[0]
		if !inputMap.IsWhollyKnown() {
			return cty.UnknownVal(retType), nil
		}

		outputMap := make(map[string]cty.Value)
		tmpMap := make(map[string][]string)

		for it := inputMap.ElementIterator(); it.Next(); {
			inKey, inVal := it.Element()
			for iter := inVal.ElementIterator(); iter.Next(); {
				_, val := iter.Element()
				if !val.Type().Equals(cty.String) {
					return cty.MapValEmpty(cty.List(cty.String)), errors.New("input must be a map of lists of strings")
				}

				outKey := val.AsString()
				if _, ok := tmpMap[outKey]; !ok {
					tmpMap[outKey] = make([]string, 0)
				}
				outVal := tmpMap[outKey]
				outVal = append(outVal, inKey.AsString())
				sort.Strings(outVal)
				tmpMap[outKey] = outVal
			}
		}

		for outKey, outVal := range tmpMap {
			values := make([]cty.Value, 0)
			for _, v := range outVal {
				values = append(values, cty.StringVal(v))
			}
			outputMap[outKey] = cty.ListVal(values)
		}

		if len(outputMap) == 0 {
			return cty.MapValEmpty(cty.List(cty.String)), nil
		}

		return cty.MapVal(outputMap), nil
	},
})
