// Code generated by mro2go testdata/pipeline_stages.mro; DO NOT EDIT.

package main

import (
	"bytes"
	"encoding/json"

	"github.com/martian-lang/martian/martian/core"
)

//
// SUM_SQUARES
//

// A structure to encode and decode args to the SUM_SQUARES stage.
type SumSquaresArgs struct {
	// The values to sum over
	//
	// These values are squared and then summed over.
	Values []float64 `json:"values"`
}

// CallName returns the name of this stage as defined in the .mro file.
func (*SumSquaresArgs) CallName() string {
	return "SUM_SQUARES"
}

// MroFileName returns the name of the .mro file which defines this stage.
func (*SumSquaresArgs) MroFileName() string {
	return "testdata/pipeline_stages.mro"
}

// A structure to encode and decode outs from the SUM_SQUARES stage.
type SumSquaresOuts struct {
	// The sum of the squares of the values
	Sum float64 `json:"sum"`
}

// A structure to encode chunk definitions for SUM_SQUARES.
// Defines the resources and chunk-specific arguments.
type SumSquaresChunkDef struct {
	*core.JobResources `json:",omitempty"`
	Value              float64 `json:"value"`
}

func (def *SumSquaresChunkDef) ArgsMap() (core.LazyArgumentMap, error) {
	m := make(core.LazyArgumentMap, 1)
	if b, err := json.Marshal(def.Value); err != nil {
		return m, err
	} else {
		m["value"] = b
	}
	return m, nil
}

func (def *SumSquaresChunkDef) ToChunkDef() (*core.ChunkDef, error) {
	args, err := def.ArgsMap()
	return &core.ChunkDef{
		Resources: def.JobResources,
		Args:      args,
	}, err
}

// A structure to decode args to the chunks for SUM_SQUARES
type SumSquaresChunkArgs struct {
	SumSquaresChunkDef
	SumSquaresArgs
}

// A structure to decode args to the join method for SUM_SQUARES
type SumSquaresJoinArgs struct {
	core.JobResources
	SumSquaresArgs
}

// A structure to encode outs from the chunks for SUM_SQUARES.
type SumSquaresChunkOuts struct {
	SumSquaresOuts
	Square float64 `json:"square"`
}

func (def *SumSquaresChunkOuts) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteRune('{')
	if b, err := json.Marshal(&def.Sum); err != nil {
		return nil, err
	} else {
		buf.WriteString("\"sum\":")
		buf.Write(b)
	}
	if b, err := json.Marshal(&def.Square); err != nil {
		return nil, err
	} else {
		buf.WriteString(",\"square\":")
		buf.Write(b)
	}
	buf.WriteRune('}')
	return buf.Bytes(), nil
}

//
// REPORT
//

// A structure to encode and decode args to the REPORT stage.
type ReportArgs struct {
	Values []float64 `json:"values"`
	Sum    float64   `json:"sum"`
}

// CallName returns the name of this stage as defined in the .mro file.
func (*ReportArgs) CallName() string {
	return "REPORT"
}

// MroFileName returns the name of the .mro file which defines this stage.
func (*ReportArgs) MroFileName() string {
	return "testdata/pipeline_stages.mro"
}

// A structure to encode and decode outs from the REPORT stage.
type ReportOuts struct {
}
