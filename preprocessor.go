package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
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
func preprocess(src string, filename string) (string, []FileLoc) {
	filebase := filepath.Base(filename)
	filedir := filepath.Dir(filename)

	// Locmap tracks original filenames and line numbers and captures
	// the source insertion mechanics.
	locmap := make([]FileLoc, lineCount(src))
	for i, _ := range locmap {
		locmap[i] = FileLoc{filebase, i}
	}
	insertOffset := 0

	// Replace all @include statements with contents of files they refer to.
	re := regexp.MustCompile("@include \"([^\\\"]+)\"")
	offsets := re.FindAllStringIndex(src, -1)
	processedSrc := re.ReplaceAllStringFunc(src, func(match string) string {
		// Get the source to be included.
		includeFilename := filepath.Join(filedir, re.FindStringSubmatch(match)[1])
		data, _ := ioutil.ReadFile(includeFilename)
		includeSrc := string(data)

		// Determine line number of src to insert included source.
		includeLine := lineNumOfOffset(src, offsets[0][0]) + insertOffset
		offsets = offsets[1:] // shift()

		// Recursively preprocess the included source.
		processedIncludeSrc, processedIncludeLocmap := preprocess(includeSrc, includeFilename)
		processedIncludeLineCount := lineCount(processedIncludeSrc)

		// Keep track of hwo much we need to increment insertion points as
		// we linearly insert more included source blocks.
		insertOffset += processedIncludeLineCount - 1 // because we're replacing 1 line with many

		// Mirror the actual source insertion in the locmap.
		locmap = append(locmap[:includeLine], append(processedIncludeLocmap, locmap[includeLine+1:]...)...)

		return processedIncludeSrc
	})
	return processedSrc, locmap
}