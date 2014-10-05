//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// MRO preprocessor.
//
package core

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

//
// Preprocessor
//
type FileLoc struct {
	fname string
	loc   int
}

func lineCount(src string) int {
	return len(strings.Split(src, "\n"))
}

func lineNumOfOffset(src string, offset int) int {
	// Converts a character offset in text into a line number.
	return strings.Count(src[0:offset], "\n")
}

func printSourceMap(src string, locmap []FileLoc) {
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		fmt.Println(locmap[i].fname, locmap[i].loc+1, line)
	}
}

/*
 * Inject contents of included files, recursively.
 */
func preprocess(src string, fname string, incPaths []string) (string, []FileLoc, *PreprocessError) {
	// Locmap tracks original filenames and line numbers and captures
	// the source insertion mechanics.
	locmap := make([]FileLoc, lineCount(src))
	for i, _ := range locmap {
		locmap[i] = FileLoc{fname, i}
	}
	insertOffset := 0

	// Replace all @include statements with contents of files they refer to.
	re := regexp.MustCompile("@include\\s+\"([^\\\"]+)\"")
	offsets := re.FindAllStringIndex(src, -1)
	fileNotFoundError := &PreprocessError{[]string{}}
	processedSrc := re.ReplaceAllStringFunc(src, func(match string) string {
		// Get name of file to be included.
		ifname := re.FindStringSubmatch(match)[1]
		if ifname == fname {
			fileNotFoundError.files = append(fileNotFoundError.files, "Cannot include self.")
			return ""
		}

		// Search incPaths for the file.
		// If not found, add this file to error list.
		ifpath, found := searchPaths(ifname, incPaths)
		if !found {
			fileNotFoundError.files = append(fileNotFoundError.files, ifname)
			return ""
		}

		// Open the file to be included.
		data, _ := ioutil.ReadFile(ifpath)
		includeSrc := string(data)

		// Determine line number of src to insert included source.
		includeLine := lineNumOfOffset(src, offsets[0][0]) + insertOffset
		offsets = offsets[1:] // shift()

		// Recursively preprocess the included source.
		processedIncludeSrc, processedIncludeLocmap, err := preprocess(includeSrc, ifname, incPaths)
		if err != nil {
			fileNotFoundError.files = append(fileNotFoundError.files, err.files...)
		}
		processedIncludeLineCount := lineCount(processedIncludeSrc)

		// Keep track of hwo much we need to increment insertion points as
		// we linearly insert more included source blocks.
		insertOffset += processedIncludeLineCount - 1 // because we're replacing 1 line with many

		// Mirror the actual source insertion in the locmap.
		locmap = append(locmap[:includeLine], append(processedIncludeLocmap, locmap[includeLine+1:]...)...)

		return processedIncludeSrc
	})
	if len(fileNotFoundError.files) > 0 {
		return processedSrc, locmap, fileNotFoundError
	}
	return processedSrc, locmap, nil
}
