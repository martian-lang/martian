//go:generate mro2go -p main -o types.go ../sum_squares.mro

package main

import (
	"fmt"
	"martian/adapter"
	"martian/core"
)

const __MRO__ = `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
) split using (
    in  float   value,
    out float   square,
)
`

// Make a chunk for each value.
func split(metadata *core.Metadata) (*core.StageDefs, error) {
	var args SumSquaresArgs
	if err := metadata.ReadInto(core.ArgsFile, &args); err != nil {
		return nil, err
	}
	sd := &core.StageDefs{
		ChunkDefs: make([]*core.ChunkDef, 0, len(args.Values)),
		JoinDef: &core.JobResources{
			Threads: 1,
			MemGB:   1,
		},
	}
	for _, val := range args.Values {
		sd.ChunkDefs = append(sd.ChunkDefs, (&SumSquaresChunkDef{
			Value: val,
			JobResources: &core.JobResources{
				Threads: 1,
				MemGB:   1,
			},
		}).ToChunkDef())
	}
	return sd, nil
}

func chunk(metadata *core.Metadata) (interface{}, error) {
	var args SumSquaresChunkArgs
	if err := metadata.ReadInto(core.ArgsFile, &args); err != nil {
		return nil, err
	} else if err := metadata.WriteRaw(core.ProgressFile, fmt.Sprintf(
		"Running with %d threads and %dGB of memory.",
		args.Threads, args.MemGB)); err != nil {
		return nil, err
	} else if err := metadata.UpdateJournal(core.ProgressFile); err != nil {
		return nil, err
	}
	return &SumSquaresChunkOuts{Square: args.Value * args.Value}, nil
}

func join(metadata *core.Metadata) (interface{}, error) {
	chunkOuts := make([]SumSquaresChunkOuts, 0, 3)
	if err := metadata.ReadInto(core.ChunkOutsFile, &chunkOuts); err != nil {
		return nil, err
	}
	var sum float64
	for _, out := range chunkOuts {
		sum += out.Square
	}
	return &SumSquaresOuts{Sum: sum}, nil
}

// Note here that a single main function handles all 3 phases for the stage.
func main() {
	adapter.RunStage(split, chunk, join)
}
