//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian runtime. This is where the action happens.
//

package syntax

import (
	"encoding/json"
	"fmt"
)

type StageCodeType int

const (
	UnknownStageLang StageCodeType = iota
	PythonStage
	ExecStage
	CompiledStage
)

func (self StageCodeType) String() string {
	switch self {
	case PythonStage:
		return "Python"
	case ExecStage:
		return "Executable"
	case CompiledStage:
		return "Compiled"
	default:
		return ""
	}
}

const (
	abr_python   = "py"
	abr_exec     = "exec"
	abr_compiled = "comp"
)

func (lang StageLanguage) Parse() (StageCodeType, error) {
	switch lang {
	case abr_python:
		return PythonStage, nil
	case abr_exec:
		return ExecStage, nil
	case abr_compiled:
		return CompiledStage, nil
	default:
		return UnknownStageLang, fmt.Errorf("Unknown language %v", lang)
	}
}

func (self StageCodeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(self.String())
}

func (self *StageCodeType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "Python":
		*self = PythonStage
	case "Executable":
		*self = ExecStage
	case "Compiled":
		*self = CompiledStage
	default:
		*self = UnknownStageLang
	}
	return nil
}
