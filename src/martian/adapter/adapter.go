//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//
// Martian golang job adapter.
//
// This package defines utility methods useful for stage code written in Go.
//
// A stage's main() function should not need to do anything besides call
// RunStage(split, chunk, join).  One executable handles all 3 phases.
//
// Stage code should NEVER directly write to the log, errors, or assert files
// through the metadata object, but should instead return an error.  For an
// assertion error, use the StageAssertion method.  For logging, use
// core.LogInfo and friends.

package adapter

import (
	"fmt"
	"martian/core"
	"martian/util"
	"os"
	"path"
	"runtime"
)

type stageAssertion struct {
	message string
}

// Get the error message.
func (self *stageAssertion) Error() string {
	return fmt.Sprintf("ASSERT:%s", self.message)
}

// Creates an error that is interpreted as an assertion.
func StageAssertion(message string) error {
	return &stageAssertion{message: message}
}

// Tests whether the given error is an assertion.
func IsAssertion(err error) bool {
	_, ok := err.(*stageAssertion)
	return ok
}

// A function for a stage's split phase.  Must return a StageDefs object.
// Stage Args, jobinfo, and so on can be read with metadata.ReadInto().
type SplitFunc func(metadata *core.Metadata) (*core.StageDefs, error)

// A function for a stage's chunk or join phase.  The returned object, if any,
// is saved to the stage _outs.  Stage args, jobinfo, and so on can be read
// with metadata.ReadInto().
type MainFunc func(metadata *core.Metadata) (interface{}, error)

// Write stage progress information.  This information will be bubbled
// up to the mrp log, unless it is overwritten by a more recent update
// first.
func UpdateProgress(metadata *core.Metadata, message string) error {
	if err := metadata.WriteRaw(core.ProgressFile, message); err != nil {
		return err
	}
	return metadata.UpdateJournal(core.ProgressFile)
}

// Parses the command line and stage inputs, runs the appropriate given stage
// code, and saves the outputs.  split and join may be nil if the stage does
// not split.
func RunStage(split SplitFunc, main MainFunc, join MainFunc) {
	util.LogTeeWriter(os.NewFile(3, "martian://log"))
	errorFile := os.NewFile(4, "martian://errors")
	// Capture panic stacks into the _errors file and exit when this method is complete.
	defer func() {
		if r := recover(); r != nil {
			var buf [8000]byte
			stack := buf[:runtime.Stack(buf[:], true)]
			fmt.Fprintf(errorFile, "Stage code panic: %v\n\n%s", r, stack)
		}
		errorFile.Close()
		os.Exit(0)
	}()
	metadata, runType := parseCommandLine()
	switch runType {
	case "split":
		runSplit(split, metadata, errorFile)
	case "main":
		runMain(main, metadata, errorFile)
	case "join":
		runMain(join, metadata, errorFile)
	default:
		fmt.Fprintf(errorFile, "ASSERT:Invalid run type %s", runType)
		return
	}
}

func parseCommandLine() (*core.Metadata, string) {
	if len(os.Args) < 5 {
		panic("Insufficient arguments.\n" +
			"Expected: <exe> [exe args...] <split|main|join> " +
			"<metadata_path> <files_path> <journal_prefix>")
	}
	args := os.Args[len(os.Args)-4:]
	runType := args[0]
	metadataPath := args[1]
	filesPath := args[2]
	fqname := path.Base(args[3])
	journalPath := path.Dir(args[3])
	return core.NewMetadataRunWithJournalPath(
			fqname, metadataPath, filesPath, journalPath, runType),
		runType
}

func runSplit(split SplitFunc, metadata *core.Metadata, errorFile *os.File) {
	if stageDefs, err := split(metadata); err != nil {
		errorFile.Write([]byte(err.Error()))
	} else if stageDefs == nil {
		errorFile.Write([]byte("Split returned nil."))
	} else {
		if err := metadata.Write(core.StageDefsFile, stageDefs); err != nil {
			fmt.Fprintf(errorFile, "Error writing stage defs: %v", err)
		} else {
			if err := metadata.UpdateJournal(core.StageDefsFile); err != nil {
				util.LogError(err, "adapter", "Error writing journal")
			}
		}
	}
}

func runMain(main MainFunc, metadata *core.Metadata, errorFile *os.File) {
	if outs, err := main(metadata); err != nil {
		errorFile.Write([]byte(err.Error()))
	} else if outs != nil {
		if err := metadata.Write(core.OutsFile, outs); err != nil {
			fmt.Fprintf(errorFile, "Error writing outs: %v", err)
		} else {
			if err := metadata.UpdateJournal(core.OutsFile); err != nil {
				util.LogError(err, "adapter", "Error writing journal")
			}
		}
	}
}
