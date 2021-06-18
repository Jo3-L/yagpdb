package templates

import (
	"reflect"
	"testing"
	"time"

	"github.com/jonas747/discordgo"
)

func ptrTo(i interface{}) interface{} {
	return &i
}

func TestSort(t *testing.T) {
	t0 := time.Now()
	t1 := t0.Add(-5 * time.Minute)
	t2 := t1.Add(2 * time.Minute)

	map0 := map[string]interface{}{"hello": "world"}
	map1 := map[string]interface{}{"bye": "world"}
	map3 := map[string]interface{}{"no": "asdfasdfasdf"}
	map4 := map[string]interface{}{"key1": "a", "key2": "b"}

	slice0 := []interface{}{"hello", "world"}
	slice1 := []interface{}{"bye", "world"}
	slice2 := []interface{}{"asdfasdf", "asdfasdf"}
	slice3 := []interface{}{"h"}

	user := discordgo.User{}

	cases := []struct {
		name      string
		slice     interface{}
		opts      []interface{}
		want      interface{}
		expectErr bool
	}{
		// no options
		{"strings", []interface{}{"a", "b", "d", "c"}, nil, []interface{}{"a", "b", "c", "d"}, false},
		{"ints", []interface{}{1, 3, 2, 4}, nil, []interface{}{1, 2, 3, 4}, false},
		{"floats", []interface{}{1.5, 0.5, 0.1, 2.5, 1.4}, nil, []interface{}{0.1, 0.5, 1.4, 1.5, 2.5}, false},
		{"uints", []interface{}{uint(5), uint(3), uint(2), uint(7)}, nil, []interface{}{uint(2), uint(3), uint(5), uint(7)}, false},
		{"slices/arrays", []interface{}{[...]string{"a", "b", "C"}, []string{"a", "b"}}, nil, []interface{}{[]string{"a", "b"}, [...]string{"a", "b", "C"}}, false},
		{"maps", []interface{}{map[string]string{"a": "b", "c": "d"}, map[interface{}]interface{}{"a": "c"}}, nil,
			[]interface{}{map[interface{}]interface{}{"a": "c"}, map[string]string{"a": "b", "c": "d"}}, false},
		{"times", []interface{}{t0, t1, t2}, nil, []interface{}{t1, t2, t0}, false},
		{"empty", []interface{}{}, nil, []interface{}{}, false},

		// reverse
		{"strings reverse", []interface{}{"a", "b", "d", "c"}, []interface{}{"reverse", true}, []interface{}{"d", "c", "b", "a"}, false},
		{"floats reverse", []interface{}{1, 3, 2, 4}, []interface{}{"reverse", true}, []interface{}{4, 3, 2, 1}, false},
		{"slices/arrays reverse", []interface{}{[...]string{"a", "b", "C"}, []string{"a", "b"}, [...]string{"a", "b", "c", "d"}}, []interface{}{"reverse", true},
			[]interface{}{[...]string{"a", "b", "c", "d"}, [...]string{"a", "b", "C"}, []string{"a", "b"}}, false},

		// stable
		{"maps stable", []interface{}{map4, map3, map1, map0}, []interface{}{"stable", true}, []interface{}{map3, map1, map0, map4}, false},

		// with key
		{"map keys with indexed type string", []interface{}{SDict{"name": "Joe"}, SDict{"name": "Asdf"}, SDict{"name": "Chloe"}}, []interface{}{"key", ptrTo("name")},
			[]interface{}{SDict{"name": "Asdf"}, SDict{"name": "Chloe"}, SDict{"name": "Joe"}}, false},
		{"map keys with indexed type int", []interface{}{SDict{"age": 100}, SDict{"age": 500}, SDict{"age": 250}, SDict{"age": 270}, SDict{"age": 50}},
			[]interface{}{"key", "age"}, []interface{}{SDict{"age": 50}, SDict{"age": 100}, SDict{"age": 250}, SDict{"age": 270}, SDict{"age": 500}}, false},
		{"map keys with indexed type mix of int values and ptrs", []interface{}{SDict{"age": ptrTo(100)}, SDict{"age": 500}, SDict{"age": ptrTo(250)}, SDict{"age": 270}, SDict{"age": ptrTo(50)}},
			[]interface{}{"key", "age"}, []interface{}{SDict{"age": ptrTo(50)}, SDict{"age": ptrTo(100)}, SDict{"age": ptrTo(250)}, SDict{"age": 270}, SDict{"age": 500}}, false},
		{"slice index with indexed type string", []interface{}{Slice{"asdf"}, Slice{"bsd"}, Slice{"abdf"}, Slice{"dddd"}, Slice{"cccc"}}, []interface{}{"key", 0},
			[]interface{}{Slice{"abdf"}, Slice{"asdf"}, Slice{"bsd"}, Slice{"cccc"}, Slice{"dddd"}}, false},

		// categorize types
		{"categorize types with element type int/string", []interface{}{"joe", "bob", 2, "abby", 4, 3}, []interface{}{"categorizeTypes", true},
			&categorizedSortResult{Ints: Slice{2, 3, 4}, Strings: Slice{"abby", "bob", "joe"}}, false},
		{"categorize types with element type int/float", []interface{}{-1, 3, 4, 1, 2, -1.5, 3.5, 4.5, 1.5}, []interface{}{"categorizeTypes", true},
			&categorizedSortResult{Ints: Slice{-1, 1, 2, 3, 4}, Floats: Slice{-1.5, 1.5, 3.5, 4.5}}, false},
		{"categorize types with element type map/slice/int", []interface{}{SDict{"key": "value"}, SDict{}, Slice{"a", "b"}, Slice{}, 3, 1}, []interface{}{"categorizeTypes", true},
			&categorizedSortResult{SlicesOrArrays: Slice{Slice{}, Slice{"a", "b"}}, Maps: Slice{SDict{}, SDict{"key": "value"}}, Ints: Slice{1, 3}}, false},
		{"categorize types with element type uint/time", []interface{}{t0, uint(3), t1, uint(2), t2, uint(1)}, []interface{}{"categorizeTypes", true},
			&categorizedSortResult{Times: Slice{t1, t2, t0}, Uints: Slice{uint(1), uint(2), uint(3)}}, false},
		{"categorize types reverse", []interface{}{Slice{"a", "b"}, Slice{}, Slice{"a", "b", "c"}, 3, 1, 2}, []interface{}{"categorizeTypes", true, "reverse", true},
			&categorizedSortResult{SlicesOrArrays: Slice{Slice{"a", "b", "c"}, Slice{"a", "b"}, Slice{}}, Ints: Slice{3, 2, 1}}, false},
		{"categorize types stable", []interface{}{map4, map3, map1, map0, slice1, slice2, slice0, slice3}, []interface{}{"categorizeTypes", true, "stable", true},
			&categorizedSortResult{Maps: Slice{map3, map1, map0, map4}, SlicesOrArrays: Slice{slice3, slice1, slice2, slice0}}, false},
		{"categorize types empty", []interface{}{}, []interface{}{"categorizeTypes", true}, &categorizedSortResult{}, false},

		// pointers to values
		{"string ptrs", []interface{}{ptrTo("hi"), ptrTo("abc"), ptrTo("dc")}, nil, []interface{}{ptrTo("abc"), ptrTo("dc"), ptrTo("hi")}, false},
		{"mix of int values and ptrs", []interface{}{ptrTo(5), ptrTo(1), ptrTo(7), 8, 6}, nil, []interface{}{ptrTo(1), ptrTo(5), 6, ptrTo(7), 8}, false},

		// misc
		{"maintains original wrapper type", Slice{"b", "c", "a"}, nil, Slice{"a", "b", "c"}, false},

		// errors
		{"not slice/array", 123, nil, nil, true},
		{"mixed types int and float", []interface{}{1, 1.5, 2, 2.5}, nil, nil, true},
		{"mixed types time and map", []interface{}{t0, map[string]interface{}{"hello": "world"}}, nil, nil, true},
		{"unsupported type 01", []interface{}{user}, nil, nil, true},
		{"unsupported type 02", []interface{}{1, user}, nil, nil, true},
		{"indexing int", []interface{}{1, 2, 3, 4, 5}, []interface{}{"key", 1}, nil, true},
		{"indexing slice with string key", []interface{}{Slice{1, 2}, Slice{3, 4}}, []interface{}{"key", "hello"}, nil, true},
		{"indexing slice key out of range 01", []interface{}{Slice{}}, []interface{}{"key", 0}, nil, true},
		{"indexing slice key out of range 02", []interface{}{Slice{}}, []interface{}{"key", -1}, nil, true},
		{"indexing map key not present in all", []interface{}{SDict{"key": 1}, SDict{"key": 2}, SDict{"keyee": 3}}, []interface{}{"key", "key"}, nil, true},
		{"indexing map key not assignable", []interface{}{SDict{"key": 1}, SDict{"key": 2}, SDict{"key": 4}}, []interface{}{"key", 0xbeef}, nil, true},
		{"indexing slice resulting in unsupported type 01", []interface{}{SDict{"key": user}}, []interface{}{"key", "key"}, nil, true},
		{"indexing slice resulting in unsupported type 02", []interface{}{SDict{"key": 1}, SDict{"key": user}}, []interface{}{"key", "key"}, nil, true},
		{"indexed value not compatible type", []interface{}{SDict{"key": 1}, SDict{"key": 1.2}}, []interface{}{"key", "key"}, nil, true},
		{"supplying key + categorizeTypes", []interface{}{}, []interface{}{"key", "asdf", "categorizeTypes", true}, nil, true},
		{"unsupported type categorizeTypes on", []interface{}{SDict{"key": "value"}, 1, 2, 3, 1.5, user}, []interface{}{"categorizeTypes", true}, nil, true},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			result, err := tmplSort(cs.slice, cs.opts...)
			if cs.expectErr {
				if err == nil {
					t.Error("wanted error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %s", err)
				return
			}

			if !reflect.DeepEqual(result, cs.want) {
				t.Errorf("unexpected result, values are not the same, got %v expected %v", result, cs.want)
			}
		})
	}
}
