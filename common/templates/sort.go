package templates

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"
)

var (
	timeType          = reflect.TypeOf((*time.Time)(nil)).Elem()
	templateSliceType = reflect.TypeOf((*Slice)(nil)).Elem()
)

func tmplSort(input interface{}, rest ...interface{}) (interface{}, error) {
	rv, _ := indirect(reflect.ValueOf(input))
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		// ok
	default:
		return nil, fmt.Errorf("cannot sort value of type %s; only slices and arrays are supported", rv.Type())
	}

	opts, err := constructSortOptsFromArgs(rest...)
	if err != nil {
		return nil, err
	}

	if opts.CategorizeTypes {
		categorized, err := categorizeByTypes(rv)
		if err != nil {
			return nil, err
		}

		rs := &reflectSliceOrArraySorter{}
		var sorter sort.Interface = rs
		if opts.Reverse {
			sorter = sort.Reverse(sorter)
		}

		for typ, vals := range categorized {
			rs.rv = vals
			rs.by = lessFuncs[typ]

			if opts.Stable {
				sort.Stable(sorter)
			} else {
				sort.Sort(sorter)
			}
		}

		return categorizedSortResultFromMap(categorized), nil
	}

	// getLessFuncFor needs at least one element to determine the type,
	// so return early if the slice/array is empty.
	if rv.Len() == 0 {
		return rv.Interface(), nil
	}

	lessFunc, err := getLessFuncStrict(rv, opts)
	if err != nil {
		return nil, err
	}

	var sorter sort.Interface = &reflectSliceOrArraySorter{rv, lessFunc}
	if opts.Reverse {
		sorter = sort.Reverse(sorter)
	}

	if opts.Stable {
		sort.Stable(sorter)
	} else {
		sort.Sort(sorter)
	}
	return rv.Interface(), nil
}

// categorizeByTypes categorizes the elements in the slice or array v by their comparable type.
// It returns a map of comparable type to a reflection value of a slice of values of that type.
func categorizeByTypes(v reflect.Value) (map[comparableType]reflect.Value, error) {
	res := make(map[comparableType]reflect.Value)
	for i := 0; i < v.Len(); i++ {
		val, _ := indirect(v.Index(i))
		typ := comparableTypeOf(val)
		if typ == typeInvalid {
			return nil, fmt.Errorf("unsupported element type %s", val.Type())
		}

		vals, ok := res[typ]
		if !ok {
			vals = reflect.MakeSlice(templateSliceType, 0, 0)
		}

		res[typ] = reflect.Append(vals, val)
	}

	return res, nil
}

// categorizedSortResult holds slices of different comparable types.
// It is returned when the CategorizeTypes option is enabled in the sort options.
type categorizedSortResult struct {
	Ints, Uints, Floats, Strings, Times, SlicesOrArrays, Maps Slice
}

func categorizedSortResultFromMap(m map[comparableType]reflect.Value) *categorizedSortResult {
	res := &categorizedSortResult{}
	for typ, rv := range m {
		sliceVal := rv.Interface().(Slice)
		switch typ {
		case typeInt:
			res.Ints = sliceVal
		case typeUint:
			res.Uints = sliceVal
		case typeFloat:
			res.Floats = sliceVal
		case typeString:
			res.Strings = sliceVal
		case typeTime:
			res.Times = sliceVal
		case typeSliceOrArray:
			res.SlicesOrArrays = sliceVal
		case typeMap:
			res.Maps = sliceVal
		}
	}

	return res
}

// getLessFuncStrict returns a less func appropriate for comparing elements in the slice or array v.
// v must have Len() > 0; otherwise, getLessFuncStrict panics.
// For getLessFuncStrict to return a valid less func, all the elements in the slice or array must
// have the same comparable type. Furthermore, if there is a key set in the options, all elements
// must be indexable by that key. The resulting indexed values must have the same comparable type as well.
// If the above criteria is not met, getLessFuncStrict returns nil.
func getLessFuncStrict(v reflect.Value, opts *sortOptions) (lessFunc, error) {
	// compute the element and indexed value type of the first value in the slice/array.
	firstV, _ := indirect(v.Index(0))
	elemT := comparableTypeOf(firstV)
	if elemT == typeInvalid {
		return nil, fmt.Errorf("unsupported element type %s", firstV.Type())
	}

	indexedT := typeInvalid
	if opts.Key.IsValid() {
		indexedV, err := safeIndex(firstV, elemT, opts.Key)
		if err != nil {
			return nil, err
		}

		indexedV, _ = indirect(indexedV)
		indexedT = comparableTypeOf(indexedV)
		if indexedT == typeInvalid {
			return nil, fmt.Errorf("unsupported indexed type %s", indexedV.Type())
		}
	}

	// compare element and indexed value types for all other values in the slice/array to those of the first element.
	for i := 1; i < v.Len(); i++ {
		curV, _ := indirect(v.Index(i))
		curElemT := comparableTypeOf(curV)
		if curElemT == typeInvalid {
			return nil, fmt.Errorf("unsupported element type %s", curV.Type())
		}

		if curElemT != elemT {
			return nil, fmt.Errorf("elements must be of the same type; found incompatible types %s and %s", curV.Type(), firstV.Type())
		}

		if opts.Key.IsValid() {
			indexedV, err := safeIndex(curV, curElemT, opts.Key)
			if err != nil {
				return nil, err
			}

			indexedV, _ = indirect(indexedV)
			curIndexedT := comparableTypeOf(indexedV)
			if curIndexedT == typeInvalid {
				return nil, fmt.Errorf("unsupported indexed type %s", indexedV.Type())
			}

			if curIndexedT != indexedT {
				return nil, errors.New("indexed values must be of the same type")
			}
		}
	}

	// no key, so use the less func for the element type.
	if !opts.Key.IsValid() {
		return lessFuncs[elemT], nil
	}

	// if there's a key set, the less func should first index the value,
	// and then use the less func for the indexed type to compare the indexed values.
	indexedLessF := lessFuncs[indexedT]
	switch elemT {
	case typeSliceOrArray:
		// no need to check whether idx is valid because we already did that when we scanned the array/slice above.
		idx := int(opts.Key.Int())
		return func(v0, v1 reflect.Value) bool {
			vv0, _ := indirect(v0.Index(idx))
			vv1, _ := indirect(v1.Index(idx))
			return indexedLessF(vv0, vv1)
		}, nil
	case typeMap:
		// see above; these operations should always be valid.
		return func(v0, v1 reflect.Value) bool {
			vv0, _ := indirect(v0.MapIndex(opts.Key))
			vv1, _ := indirect(v1.MapIndex(opts.Key))
			return indexedLessF(vv0, vv1)
		}, nil
	}

	// opts.Key.IsValid() should only be true at this point if the element type is indexable;
	// so elemT should only ever be typeSliceOrArray or typeMap.
	panic("unreachable")
}

// safeIndex indexes v by k, returning the resulting value or an error.
func safeIndex(v reflect.Value, vType comparableType, k reflect.Value) (reflect.Value, error) {
	switch vType {
	case typeSliceOrArray:
		switch k.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// ok
		default:
			return reflect.Value{}, fmt.Errorf("when sorting collections of slices/arrays, key must be an int type; got %s instead", k.Type())
		}

		// check bounds
		idx := int(k.Int())
		if idx < 0 || idx >= v.Len() {
			return reflect.Value{}, errors.New("key out of range")
		}

		return v.Index(idx), nil
	case typeMap:
		// check assignability
		if !k.Type().AssignableTo(v.Type().Key()) {
			return reflect.Value{}, fmt.Errorf("key type (%s) is not assignable to map key type (%s)", k.Type(), v.Type().Key())
		}

		vv := v.MapIndex(k)
		if !vv.IsValid() {
			return reflect.Value{}, errors.New("invalid key")
		}
		return vv, nil
	}

	return reflect.Value{}, fmt.Errorf("cannot index value of type %s", v.Type())
}

// comparableType is the set of supported comparable types.
type comparableType uint8

const (
	typeInvalid      comparableType = iota
	typeInt                         // value's Kind is guaranteed to be Int, Int8, Int16, Int32, or Int64.
	typeUint                        // value's Kind is guaranteed to be Uint, Uint8, Uint16, Uint32, or Uint64.
	typeFloat                       // value's Kind is guaranteed to be Float32 or Float64.
	typeString                      // value's Kind is guaranteed to be String.
	typeTime                        // value's Type is guaranteed to be timeType.
	typeSliceOrArray                // value's Kind is guaranteed to be Slice or Array.
	typeMap                         // value's Kind is guaranteed to be Map.
)

type lessFunc func(reflect.Value, reflect.Value) bool

// Less funcs for supported comparable types.
var lessFuncs = map[comparableType]lessFunc{
	typeInt:          func(v0, v1 reflect.Value) bool { return v0.Int() < v1.Int() },
	typeUint:         func(v0, v1 reflect.Value) bool { return v0.Uint() < v1.Uint() },
	typeFloat:        func(v0, v1 reflect.Value) bool { return v0.Float() < v1.Float() },
	typeString:       func(v0, v1 reflect.Value) bool { return v0.String() < v1.String() },
	typeTime:         func(v0, v1 reflect.Value) bool { return v0.Interface().(time.Time).Before(v1.Interface().(time.Time)) },
	typeSliceOrArray: func(v0, v1 reflect.Value) bool { return v0.Len() < v1.Len() },
	typeMap:          func(v0, v1 reflect.Value) bool { return v0.Len() < v1.Len() },
}

// comparableTypeOf returns the comparable type of v.
// If v is not a comparable type, it returns typeInvalid.
func comparableTypeOf(v reflect.Value) comparableType {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return typeInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return typeUint
	case reflect.Float32, reflect.Float64:
		return typeFloat
	case reflect.String:
		return typeString
	case reflect.Struct:
		if v.Type() == timeType {
			return typeTime
		}
	case reflect.Slice, reflect.Array:
		return typeSliceOrArray
	case reflect.Map:
		return typeMap
	}

	return typeInvalid
}

// reflectSliceOrArraySorter is an implementation of sort.Interface for reflection values of kind Slice or Array.
type reflectSliceOrArraySorter struct {
	rv reflect.Value
	by lessFunc
}

func (rs *reflectSliceOrArraySorter) Len() int {
	return rs.rv.Len()
}

func (rs *reflectSliceOrArraySorter) Swap(i, j int) {
	v := rs.rv.Index(i).Interface()
	rs.rv.Index(i).Set(rs.rv.Index(j))
	rs.rv.Index(j).Set(reflect.ValueOf(v))
}

func (rs *reflectSliceOrArraySorter) Less(i, j int) bool {
	v0, _ := indirect(rs.rv.Index(i))
	v1, _ := indirect(rs.rv.Index(j))
	return rs.by(v0, v1)
}

type sortOptions struct {
	Reverse         bool          // Reverse specifies whether greater values should come first.
	Stable          bool          // Stable specifies whether the sort should be stable.
	CategorizeTypes bool          // CategorizeType specifies whether the slice elements should be categorized by type and then sorted by type.
	Key             reflect.Value // Key is the key to index slices/arrays/maps by for comparison. The zero value denotes no key.
}

var defaultSortOptions = sortOptions{
	Reverse:         false,
	Stable:          false,
	CategorizeTypes: false,
	Key:             reflect.Value{},
}

func constructSortOptsFromArgs(args ...interface{}) (*sortOptions, error) {
	if len(args) == 0 {
		return &defaultSortOptions, nil
	}

	opts := defaultSortOptions

	dict, err := StringKeyDictionary(args...)
	if err != nil {
		return nil, err
	}

	if v, ok := dict["reverse"]; ok {
		switch t := v.(type) {
		case bool:
			opts.Reverse = t
		case *bool:
			opts.Reverse = *t
		default:
			return nil, fmt.Errorf("cannot use type %T for reverse option", v)
		}
	}

	if v, ok := dict["stable"]; ok {
		switch t := v.(type) {
		case bool:
			opts.Stable = t
		case *bool:
			opts.Stable = *t
		default:
			return nil, fmt.Errorf("cannot use type %T for stable option", v)
		}
	}

	if v, ok := dict["categorizeTypes"]; ok {
		switch t := v.(type) {
		case bool:
			opts.CategorizeTypes = t
		case *bool:
			opts.CategorizeTypes = *t
		default:
			return nil, fmt.Errorf("cannot use type %T for categorizeTypes option", v)
		}
	}

	if v, ok := dict["key"]; ok {
		if opts.CategorizeTypes {
			return nil, errors.New("cannot specify a key when categorizeTypes is enabled")
		}

		opts.Key, _ = indirect(reflect.ValueOf(v))
	}

	return &opts, nil
}
