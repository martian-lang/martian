// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package core

import (
	"encoding/json"
	"martian/syntax"
	"reflect"
	"strings"
	"testing"
)

func TestArgumentMapValidate(t *testing.T) {
	var def ChunkDef
	if err := json.Unmarshal([]byte(`{
		"__threads": 4,
		"__mem_gb": 3,
		"foo": 12,
		"bar": 1.2,
		"baz": { "fooz": "bars" },
		"bath": "soap"
	}`), &def); err != nil {
		t.Errorf("Unmarshal failure: %v", err)
	}
	plist := []syntax.Param{
		&syntax.InParam{
			Id:    "foo",
			Tname: "int",
		},
		&syntax.InParam{
			Id:    "bar",
			Tname: "float",
		},
		&syntax.InParam{
			Id:    "baz",
			Tname: "map",
		},
	}
	ptable := make(map[string]syntax.Param, len(plist))
	for _, p := range plist {
		ptable[p.GetId()] = p
	}
	params := syntax.Params{
		Table: ptable,
		List:  plist,
	}
	if err := def.Args.Validate(&params); err == nil {
		t.Errorf("Expected error from extra param, got none.")
	} else if strings.TrimSpace(err.Error()) != "Unexpected parameter 'bath'" {
		t.Errorf(
			"Validation error: expected \""+
				"Unexpected parameter 'bath'"+
				"\", got \"%v\"",
			err)
	}
	bath := &syntax.InParam{
		Id:    "bath",
		Tname: "string",
	}
	params.Table[bath.Id] = bath
	params.List = append(params.List, bath)
	if err := def.Args.Validate(&params); err != nil {
		t.Errorf("Validation error: expected success, got %v", err)
	}
	params.Table["bar"].(*syntax.InParam).Tname = "int"
	if err := def.Args.Validate(&params); err == nil {
		t.Errorf("Expected error from float, got none.")
	} else if strings.TrimSpace(err.Error()) !=
		"int parameter 'bar' with incorrect type json.Number" {
		t.Errorf(
			"Validation error: expected \""+
				"int parameter 'bar' with incorrect type json.Number"+
				"\", got \"%v\"",
			err)
	}
	params.Table["bar"].(*syntax.InParam).Tname = "float"
	missing := &syntax.InParam{
		Id:    "miss",
		Tname: "string",
	}
	params.Table[missing.Id] = missing
	params.List = append(params.List, missing)
	if err := def.Args.Validate(&params); err == nil {
		t.Errorf("Expected error from missing parameter, got none.")
	} else if strings.TrimSpace(err.Error()) != "Missing parameter 'miss'" {
		t.Errorf(
			"Validation error: expected \""+
				"Missing parameter 'miss'"+
				"\", got \"%v\"",
			err)
	}
}

type toyStruct struct {
	Iface  interface{}
	Map    map[string]int
	Int    int
	Float  float64 `json:"n"`
	String string  `json:"s,omitempty"`
	IntP   *int
}

func TestMakeArgumentMap(t *testing.T) {
	s := toyStruct{
		Int:   5,
		Float: 6,
	}
	m := MakeArgumentMap(s)
	if len(m) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(m))
	}
	check := func(m ArgumentMap, key string, value interface{}) {
		t.Helper()
		b, _ := json.Marshal(m)
		if v, ok := m[key]; !ok {
			t.Errorf("Missing key %s\t%s", key, string(b))
		} else if !reflect.DeepEqual(v, value) {
			t.Errorf("Incorrect value for %s: expected %v actual %v\n%s",
				key, value, v, string(b))
		}
	}
	check(m, "Iface", s.Iface)
	check(m, "Map", s.Map)
	check(m, "Int", s.Int)
	check(m, "n", s.Float)
	check(m, "IntP", s.IntP)
	s.String = "foo"
	m = MakeArgumentMap(s)
	check(m, "s", s.String)
	m = MakeArgumentMap(map[string]string{
		"foo": "bar",
	})
	check(m, "foo", "bar")
}

func TestArgumentMapDecode(t *testing.T) {
	check := func(expected ArgumentMap, actual interface{}) {
		t.Helper()
		if err := expected.Decode(actual); err != nil {
			t.Errorf("Error decoding: %v", err)
		}
	}
	s := toyStruct{}
	check(ArgumentMap{
		"Iface": map[string]string{"foo": "bar"},
		"s":     "baz",
	}, &s)
	if (s.Iface.(map[string]string))["foo"] != "bar" {
		t.Errorf("Incorrect foo in iface: %v", (s.Iface.(map[string]interface{}))["foo"])
	}
	if s.String != "baz" {
		t.Errorf("Incorrect String: %s", s.String)
	}
	checkMap := func(expected ArgumentMap, actual interface{}) {
		t.Helper()
		check(expected, actual)
		if be, err := json.Marshal(expected); err != nil {
			t.Errorf("Error encoding: %v", err)
		} else if ba, err := json.Marshal(actual); err != nil {
			t.Errorf("Error encoding: %v", err)
		} else if string(be) != string(ba) {
			t.Errorf("Incorrect decode: expected %s got %s", string(be), string(ba))
		}
	}
	checkMap(ArgumentMap{
		"foo": "bar",
	}, make(map[string]string))
	checkMap(ArgumentMap{
		"foo": 1,
	}, make(map[string]int))
	checkMap(ArgumentMap{
		"foo": 1,
	}, make(map[string]interface{}))
	m := ArgumentMap{
		"foo": "bar",
	}
	if err := m.Decode(make(map[string]int)); err == nil {
		t.Errorf("Expected error.")
	}
}
