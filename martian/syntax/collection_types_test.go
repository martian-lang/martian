// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

package syntax

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestArrayFilterJson(t *testing.T) {
	_, _, ast, err := ParseSourceBytes([]byte(`
struct BAR(
    string c,
)

struct BAZ(
	int a,
	BAR b,
)

struct FOO(
	int   bar,
	BAZ[] baz,
)`), "example.mro", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	lookup := &ast.TypeTable
	ty := lookup.Get(TypeId{Tname: "FOO", MapDim: 1})
	if !ty.CanFilter() {
		t.Error("should filter")
	}
	j := []byte(`{
		"foo1": {
			"bar": 0,
			"baz": [
				{
					"a": 1.0,
					"b": {
						"c": "2",
						"d": 8
					}
				},
				{
					"a": 3,
					"b": {
						"c": "4"
					},
					"c": "bar"
				}
			]
		}
	}`)
	if b, fatal, err := ty.FilterJson(j, lookup); err == nil {
		t.Error("expected warning")
	} else if fatal {
		t.Error("expected non-fatal")
	} else {
		var buf bytes.Buffer
		if err := json.Indent(&buf, b, "\t\t", "\t"); err != nil {
			t.Error(err)
		} else if s := buf.String(); s != `{
			"foo1": {
				"bar": 0,
				"baz": [
					{
						"a": 1,
						"b": {
							"c": "2"
						}
					},
					{
						"a": 3,
						"b": {
							"c": "4"
						}
					}
				]
			}
		}` {
			t.Error(s)
		}
	}
}

func TestTypedMapValidJson(t *testing.T) {
	var lookup TypeLookup
	lookup.init(2)
	js := UserType{Id: "json"}
	if err := lookup.AddUserType(&js); err != nil {
		t.Fatal(err)
	}
	ty := lookup.GetMap(&js)
	if ty == nil {
		t.Fatal("could not get map type")
	} else if ty.Elem == nil {
		t.Fatal("no element type")
	}
	check := func(s, exp string) {
		t.Helper()
		var alarms strings.Builder
		err := ty.IsValidJson(json.RawMessage([]byte(s)), &alarms, &lookup)
		if exp == "" && err == nil {
			return
		} else if exp == "" && err != nil {
			t.Error(err)
		} else if err == nil {
			t.Error("expected failure", exp)
		} else if s := err.Error(); !strings.Contains(s, exp) {
			t.Error(s)
		}
	}
	check(`{"foo": "bar"}`, "")
	check(`{"foo": ""}`, "")
	check(`{"": "bar"}`, "empty string")
	check(`{"..": "bar"}`, "reserved name")
	check(`{"a/a": "bar"}`, "'/'")
	ty = lookup.GetMap(&builtinString)
	check(`{"foo": "bar"}`, "")
	check(`{"foo": ""}`, "")
	check(`{"": "bar"}`, "")
	check(`{"..": "bar"}`, "")
	check(`{"a/a": "bar"}`, "")
	check(`{"foo": 1}`, "cannot be parsed as a string")
	check(`null`, "")
	check(`["foo"]`, "cannot be parsed as a map")
}

func TestArrayValidJson(t *testing.T) {
	var lookup TypeLookup
	lookup.init(2)
	ty := lookup.GetArray(&builtinString, 1).(*ArrayType)
	if ty == nil {
		t.Fatal("could not get array type")
	} else if ty.Elem == nil {
		t.Fatal("no element type")
	}
	check := func(s, exp string) {
		t.Helper()
		var alarms strings.Builder
		err := ty.IsValidJson(json.RawMessage([]byte(s)), &alarms, &lookup)
		if exp == "" && err == nil {
			return
		} else if exp == "" && err != nil {
			t.Error(err)
		} else if err == nil {
			t.Error("expected failure", exp)
		} else if s := err.Error(); !strings.Contains(s, exp) {
			t.Error(s)
		}
	}
	check(`["foo"]`, "")
	check(`null`, "")
	check(`[]`, "")
	check(`[1]`, "cannot be parsed as a string")
	check(`["foo", 1]`, "cannot be parsed as a string")
	check(`[null, "foo"]`, "")
	check(`[null, "foo", 1]`, "cannot be parsed as a string")
	check(`{"foo": ""}`, "cannot be parsed as an array")
	ty = lookup.GetArray(&builtinString, 2).(*ArrayType)
	if ty == nil {
		t.Fatal("could not get array type")
	} else if ty.Elem == nil {
		t.Fatal("no element type")
	}
	check(`["foo"]`, "cannot be parsed as an array")
	check(`[[]]`, "")
	check(`[]`, "")
	check(`[null]`, "")
}
