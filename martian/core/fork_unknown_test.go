package core

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/martian-lang/martian/martian/syntax"
)

// A json.Marshaler with array kind which is not one of the types handled
// explicitly by getUnknownLength.
type testFixedArray [3]int

func (a testFixedArray) MarshalJSON() ([]byte, error) {
	return json.Marshal([3]int(a))
}

// A json.Marshaler with map kind which is not one of the types handled
// explicitly by getUnknownKeys.
type testStringMap map[string]int

func (m testStringMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]int(m))
}

type testIntMap map[int]int

func (m testIntMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[int]int(m))
}

type testStringer struct{}

func (testStringer) MarshalJSON() ([]byte, error) { return []byte("1"), nil }
func (testStringer) String() string               { return "stringer" }

type testGoStringer struct{}

func (testGoStringer) MarshalJSON() ([]byte, error) { return []byte("1"), nil }
func (testGoStringer) GoString() string             { return "gostringer" }

type testPlain struct{}

func (testPlain) MarshalJSON() ([]byte, error) { return []byte("1"), nil }

func TestGetUnknownLength(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		in     json.Marshaler
		name   string
		expect int
	}{
		{name: "nil", in: nil, expect: 0},
		{name: "array exp", in: &syntax.ArrayExp{
			Value: []syntax.Exp{&syntax.IntExp{Value: 1}, &syntax.IntExp{Value: 2}},
		}, expect: 2},
		{name: "null exp", in: new(syntax.NullExp), expect: 0},
		{name: "marshaller array", in: marshallerArray{
			json.RawMessage("1"), json.RawMessage("2"), json.RawMessage("3"),
		}, expect: 3},
		{name: "raw null", in: json.RawMessage("null"), expect: 0},
		{name: "raw empty", in: json.RawMessage("[]"), expect: 0},
		{name: "raw array", in: json.RawMessage("[1,2]"), expect: 2},
		{name: "fixed array", in: testFixedArray{1, 2, 3}, expect: 3},
		{name: "pointer to fixed array", in: &testFixedArray{1, 2, 3}, expect: 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			l, err := getUnknownLength(c.in)
			if err != nil {
				t.Fatal(err)
			}
			if l != c.expect {
				t.Errorf("%d != %d", l, c.expect)
			}
		})
	}
	for name, in := range map[string]json.Marshaler{
		"stringer":   testStringer{},
		"gostringer": testGoStringer{},
		"plain":      testPlain{},
		"bad raw":    json.RawMessage("[1,"),
	} {
		t.Run("error "+name, func(t *testing.T) {
			if _, err := getUnknownLength(in); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestGetUnknownKeys(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		in     json.Marshaler
		expect []string
	}{
		{"nil", nil, nil},
		{"empty map exp", &syntax.MapExp{}, nil},
		{"map exp", &syntax.MapExp{
			Value: map[string]syntax.Exp{
				"a": &syntax.IntExp{Value: 1},
				"b": &syntax.IntExp{Value: 2},
			},
		}, []string{"a", "b"}},
		{"null exp", new(syntax.NullExp), nil},
		{"empty marshaler map", MarshalerMap{}, nil},
		{"marshaler map", MarshalerMap{
			"x": json.RawMessage("1"),
			"y": json.RawMessage("2"),
		}, []string{"x", "y"}},
		{"empty lazy map", LazyArgumentMap{}, nil},
		{"lazy map", LazyArgumentMap{
			"x": json.RawMessage("1"),
			"y": json.RawMessage("2"),
		}, []string{"x", "y"}},
		{"raw null", json.RawMessage("null"), nil},
		{"raw", json.RawMessage(`{"q":1,"r":2}`), []string{"q", "r"}},
		{"string map", testStringMap{"k": 1, "l": 2}, []string{"k", "l"}},
		{"pointer to string map", &testStringMap{"k": 1}, []string{"k"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			keys, err := getUnknownKeys(c.in)
			if err != nil {
				t.Fatal(err)
			}
			got := append([]string(nil), keys...)
			sort.Strings(got)
			if len(got) != len(c.expect) {
				t.Fatalf("%v != %v", got, c.expect)
			}
			for i, k := range got {
				if k != c.expect[i] {
					t.Errorf("%v != %v", got, c.expect)
				}
			}
		})
	}
	for name, in := range map[string]json.Marshaler{
		"int keys": testIntMap{1: 1},
		"plain":    testPlain{},
		"bad raw":  json.RawMessage("{1,"),
	} {
		t.Run("error "+name, func(t *testing.T) {
			if _, err := getUnknownKeys(in); err == nil {
				t.Error("expected error")
			}
		})
	}
}
