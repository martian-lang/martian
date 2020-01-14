// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/martian-lang/martian/martian/core"
)

func ExampleGoName() {
	for _, n := range []string{
		"STAGE_NAME",
		"_StageName",
		"_STAGE_NAME",
		"STAGE",
		"_Stage",
		"_stage",
		"paramName",
		"param_name",
		"_paramName",
		"_param_name",
	} {
		fmt.Println(n, "->", GoName(n))
	}
	// Output:
	// STAGE_NAME -> StageName
	// _StageName -> StageName
	// _STAGE_NAME -> StageName
	// STAGE -> Stage
	// _Stage -> Stage
	// _stage -> Stage
	// paramName -> ParamName
	// param_name -> ParamName
	// _paramName -> ParamName
	// _param_name -> ParamName
}

// Test that the go output is the same as what is being tested
// for functionality elsewhere.
func TestMroToGo(t *testing.T) {
	mrosrc, err := ioutil.ReadFile(path.Join("testdata", "pipeline_stages.mro"))
	if err != nil {
		t.Fatal(err)
	}
	var dest bytes.Buffer
	if err := MroToGo(&dest,
		mrosrc, "testdata/pipeline_stages.mro", "",
		nil,
		"main", "split_test.go"); err != nil {
		t.Fatal(err)
	}
	goSrc := dest.String()
	if expectedSrc, err := ioutil.ReadFile("split_test.go"); err != nil {
		t.Fatal(err)
	} else if string(expectedSrc) != goSrc {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expectedSrc, goSrc)
	}
}

func serialize(t *testing.T, obj interface{}, expected string) {
	t.Helper()
	if b, err := json.MarshalIndent(obj, "\t", "\t"); err != nil {
		t.Errorf("Error serializing: %v", err)
	} else if strings.TrimSpace(string(b)) != strings.TrimSpace(expected) {
		t.Errorf("Expected: %s\nGot: %s", expected, string(b))
	}
}

func TestSerialize(t *testing.T) {
	serialize(t, SumSquaresArgs{Values: []float64{4.0}}, `
	{
		"values": [
			4
		]
	}`)
	serialize(t, SumSquaresOuts{Sum: 5.0}, `
	{
		"sum": 5
	}`)
	serialize(t, ReportArgs{Sum: 6.0, Values: []float64{1.0, 2.0}}, `
	{
		"values": [
			1,
			2
		],
		"sum": 6
	}`)
	serialize(t, SumSquaresChunkOuts{Square: 4}, `
	{
		"sum": 0,
		"square": 4
	}`)
}

func deserialize(t *testing.T, src string, obj, expected interface{}) {
	t.Helper()
	if err := json.Unmarshal([]byte(src), obj); err != nil {
		t.Errorf("Error unmarshalling: %v", err)
	} else if !reflect.DeepEqual(obj, expected) {
		t.Errorf("Not equal: expected %v\t got %v", obj, expected)
	}
}

func TestDeserialize(t *testing.T) {
	obj := &SumSquaresChunkArgs{}
	deserialize(t, `{"values":[5,6],"value":5,"__threads":4}`,
		obj, &SumSquaresChunkArgs{
			SumSquaresArgs: SumSquaresArgs{Values: []float64{5, 6}},
			SumSquaresChunkDef: SumSquaresChunkDef{
				Value: 5,
				JobResources: &core.JobResources{
					Threads: 4,
				},
			},
		})
	if obj.Threads != 4 {
		t.Errorf("Expected 4 threads, got %d", obj.Threads)
	}
}

func TestToChunkDef(t *testing.T) {
	def := SumSquaresChunkDef{JobResources: &core.JobResources{}}
	def.Threads = 2
	def.Value = 4
	cd, err := def.ToChunkDef()
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(cd, &core.ChunkDef{
		Resources: &core.JobResources{Threads: 2},
		Args:      core.LazyArgumentMap{"value": json.RawMessage([]byte("4"))},
	}) {
		t.Errorf("Incorrect result: %v", cd)
	}
}
