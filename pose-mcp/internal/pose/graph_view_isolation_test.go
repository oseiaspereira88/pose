package pose

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

// Deep isolation of the delivery graph a cache hands out (spec
// pose-bounded-isolated-graph-caches R2). The check is structural, not a list of
// the fields someone remembered: it fills every slice, map and pointer of the graph
// with a value, copies it, and finds every reference the copy shares with the
// original.

func populate(v reflect.Value, depth int) {
	if depth > 6 {
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanSet() {
				populate(v.Field(i), depth+1)
			}
		}
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 2, 2)
		for i := 0; i < s.Len(); i++ {
			populate(s.Index(i), depth+1)
		}
		v.Set(s)
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		key := reflect.New(v.Type().Key()).Elem()
		populate(key, depth+1)
		val := reflect.New(v.Type().Elem()).Elem()
		populate(val, depth+1)
		m.SetMapIndex(key, val)
		v.Set(m)
	case reflect.Ptr:
		p := reflect.New(v.Type().Elem())
		populate(p.Elem(), depth+1)
		v.Set(p)
	case reflect.String:
		v.SetString(fmt.Sprintf("s%d", depth))
	case reflect.Int, reflect.Int64:
		v.SetInt(int64(depth) + 1)
	case reflect.Bool:
		v.SetBool(true)
	}
}

// references records the address of every slice backing array, map and pointer
// reachable from v.
func references(v reflect.Value, path string, out map[uintptr]string) {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			references(v.Field(i), path+"."+v.Type().Field(i).Name, out)
		}
	case reflect.Slice:
		if v.Len() > 0 {
			out[v.Pointer()] = path
		}
		for i := 0; i < v.Len(); i++ {
			references(v.Index(i), fmt.Sprintf("%s[%d]", path, i), out)
		}
	case reflect.Map:
		if !v.IsNil() {
			out[v.Pointer()] = path
		}
		iter := v.MapRange()
		for iter.Next() {
			references(iter.Value(), path+"{}", out)
		}
	case reflect.Ptr:
		if !v.IsNil() {
			out[v.Pointer()] = path
			references(v.Elem(), path+"*", out)
		}
	}
}

func sharedReferences(a, b any) []string {
	ra, rb := map[uintptr]string{}, map[uintptr]string{}
	references(reflect.ValueOf(a), "graph", ra)
	references(reflect.ValueOf(b), "graph", rb)
	var shared []string
	for addr, path := range ra {
		if _, ok := rb[addr]; ok {
			shared = append(shared, path)
		}
	}
	sort.Strings(shared)
	return shared
}

func TestACachedGraphSharesNoReferenceWithItsCopies(t *testing.T) {
	var graph DeliveryIntegrityGraph
	populate(reflect.ValueOf(&graph).Elem(), 0)
	copied := copyDeliveryIntegrityGraph(graph)
	if !reflect.DeepEqual(graph, copied) {
		t.Fatal("the copy does not equal the original")
	}
	if shared := sharedReferences(graph, copied); len(shared) != 0 {
		t.Fatalf("the copy shares %d references with the original:\n%v", len(shared), shared)
	}
}

func TestMutatingAReadNeverReachesTheNextRead(t *testing.T) {
	deliveryGraphCache.Reset()
	raw := []byte(`{"schema_version":1,"input_digest":"d","nodes":[{"id":"n","type":"t","attributes":{"k":"v"}}],"edges":[],"claims":[],"change_sets":[],"reverse":{"p":["a"]},"findings":[{"id":"f","path":"p"}]}`)
	first, ok := parseDeliveryIntegrityGraph("root", raw)
	if !ok {
		t.Fatal("parse failed")
	}
	first.Nodes[0].Attributes["k"] = "mutated"
	first.Nodes[0].Attributes["new"] = "x"
	first.Reverse["p"][0] = "mutated"
	first.Findings[0].Path = "mutated"
	first.Nodes = first.Nodes[:0]
	second, ok := parseDeliveryIntegrityGraph("root", raw)
	if !ok || len(second.Nodes) != 1 || second.Nodes[0].Attributes["k"] != "v" || len(second.Nodes[0].Attributes) != 1 ||
		second.Reverse["p"][0] != "a" || second.Findings[0].Path != "p" {
		t.Fatalf("a caller's write reached the next read: %+v", second)
	}
}
