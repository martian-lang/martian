package main

import (
	"martian/adapter"
	"martian/core"
)

const __MRO__ = `
stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
) split using (
    in  float   value,
)
`

type argsType struct {
	Values  []float64 `json:"values"`
	Threads int       `json:"__threads"`
	MemGB   int       `json:"__mem_gb"`
}

type outsType struct {
	Sum float64 `json:"sum"`
}

type chunkArgsType struct {
	Value   float64 `json:"value"`
	Threads int     `json:"__threads"`
	MemGB   int     `json:"__mem_gb"`
}

// Make a chunk for each value.
func split(metadata *core.Metadata) (*core.StageDefs, error) {
	var args argsType
	if err := metadata.ReadInto(core.ArgsFile, &args); err != nil {
		return nil, err
	}
	sd := &core.StageDefs{
		ChunkDefs: make([]map[string]interface{}, 0, len(args.Values)),
		JoinDef: map[string]interface{}{
			"__threads": 1,
			"__mem_gb":  1,
		},
	}
	for _, val := range args.Values {
		sd.ChunkDefs = append(sd.ChunkDefs, map[string]interface{}{
			"value":     val,
			"__threads": 1,
			"__mem_gb":  1,
		})
	}
	return sd, nil
}

func chunk(metadata *core.Metadata) (interface{}, error) {
	var args chunkArgsType
	if err := metadata.ReadInto(core.ArgsFile, &args); err != nil {
		return nil, err
	}
	return &outsType{Sum: args.Value * args.Value}, nil
}

func join(metadata *core.Metadata) (interface{}, error) {
	chunkOuts := make([]outsType, 0, 3)
	if err := metadata.ReadInto(core.ChunkOutsFile, &chunkOuts); err != nil {
		return nil, err
	}
	var sum float64
	for _, out := range chunkOuts {
		sum += out.Sum
	}
	return &outsType{Sum: sum}, nil
}

// Note here that a single main function handles all 3 phases for the stage.
func main() {
	adapter.RunStage(split, chunk, join)
}
