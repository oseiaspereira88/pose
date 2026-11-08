package pose

import "reflect"

// estimateRetainedBytes estimates the heap a value retains by walking it: the bytes
// of every string, the backing arrays of slices, the entries of maps and what a
// pointer reaches, with a fixed cost per header, element and entry. It accounts for
// every field of the type, present and future, instead of charging a constant for
// the ones someone remembered (spec pose-bounded-isolated-graph-caches R4). It is an
// estimate: TestGraphByteEstimateTracksTheMeasuredHeap calibrates it on the real
// index, and it errs high.
func estimateRetainedBytes(v any) int64 {
	return estimateValue(reflect.ValueOf(v), 0)
}

const (
	estimateHeader   = 16 // a string header, or the share of a slice or map header beyond its elements
	estimateEntry    = 16 // the bookkeeping a map entry costs beyond its key and value
	estimateMaxDepth = 32
)

// estimateValue counts a value once, with its own slot: a string is its header and
// bytes, a struct is the sum of its fields (so a flat field counts its size), a
// slice is its header and its elements, a map its header, entries, keys and values.
func estimateValue(v reflect.Value, depth int) int64 {
	if !v.IsValid() || depth > estimateMaxDepth {
		return 0
	}
	switch v.Kind() {
	case reflect.String:
		return int64(v.Len()) + estimateHeader
	case reflect.Slice:
		if v.IsNil() {
			return 8
		}
		elem := v.Type().Elem()
		if isFlat(elem) {
			return estimateHeader + int64(v.Len())*int64(elem.Size())
		}
		n := int64(estimateHeader)
		for i := 0; i < v.Len(); i++ {
			n += estimateValue(v.Index(i), depth+1)
		}
		return n
	case reflect.Map:
		if v.IsNil() {
			return 8
		}
		n := int64(estimateHeader) + 32
		iter := v.MapRange()
		for iter.Next() {
			n += estimateEntry + estimateValue(iter.Key(), depth+1) + estimateValue(iter.Value(), depth+1)
		}
		return n
	case reflect.Ptr:
		if v.IsNil() {
			return 8
		}
		return 8 + estimateValue(v.Elem(), depth+1)
	case reflect.Interface:
		if v.IsNil() {
			return 16
		}
		return 16 + estimateValue(v.Elem(), depth+1)
	case reflect.Struct:
		var n int64
		for i := 0; i < v.NumField(); i++ {
			n += estimateValue(v.Field(i), depth+1)
		}
		return n
	default:
		return int64(v.Type().Size())
	}
}

// isFlat reports whether a type holds no pointers, strings, slices or maps, so a
// slice of it is its length times the element size.
func isFlat(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	case reflect.Array:
		return isFlat(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !isFlat(t.Field(i).Type) {
				return false
			}
		}
		return true
	}
	return false
}

// deepCopyValue copies a value so that the copy shares no slice, map or pointer with
// the original. It covers every field by construction, so a field added later is
// isolated without anyone remembering it; the graph keeps a hand-written copy only
// because it is hot, and a reflection test holds that copy to the same standard.
func deepCopyValue[T any](in T) T {
	out := reflect.New(reflect.TypeOf(in)).Elem()
	copyInto(out, reflect.ValueOf(in))
	return out.Interface().(T)
}

func copyInto(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			if dst.Field(i).CanSet() {
				copyInto(dst.Field(i), src.Field(i))
			}
		}
	case reflect.Slice:
		if src.IsNil() {
			return
		}
		s := reflect.MakeSlice(src.Type(), src.Len(), src.Len())
		for i := 0; i < src.Len(); i++ {
			copyInto(s.Index(i), src.Index(i))
		}
		dst.Set(s)
	case reflect.Map:
		if src.IsNil() {
			return
		}
		m := reflect.MakeMapWithSize(src.Type(), src.Len())
		iter := src.MapRange()
		for iter.Next() {
			val := reflect.New(src.Type().Elem()).Elem()
			copyInto(val, iter.Value())
			m.SetMapIndex(iter.Key(), val)
		}
		dst.Set(m)
	case reflect.Ptr:
		if src.IsNil() {
			return
		}
		p := reflect.New(src.Type().Elem())
		copyInto(p.Elem(), src.Elem())
		dst.Set(p)
	case reflect.Interface:
		if src.IsNil() {
			return
		}
		inner := reflect.New(src.Elem().Type()).Elem()
		copyInto(inner, src.Elem())
		dst.Set(inner)
	default:
		dst.Set(src)
	}
}
