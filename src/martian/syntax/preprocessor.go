//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO preprocessor.
//
package syntax

import (
	"io/ioutil"
	"martian/util"
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

/*
 * Inject contents of included files, recursively.
 */
func preprocess(src string, fname string, incPaths []string) (string, []string, []FileLoc, *PreprocessError) {
	// Locmap tracks original filenames and line numbers and captures
	// the source insertion mechanics.
	locmap := make([]FileLoc, lineCount(src))
	for i := range locmap {
		locmap[i] = FileLoc{fname, i}
	}
	insertOffset := 0

	// Replace all @include statements with contents of files they refer to.
	re := regexp.MustCompile("@include\\s+\"([^\\\"]+)\"")
	offsets := re.FindAllStringIndex(src, -1)
	fileNotFoundError := &PreprocessError{[]string{}}
	ifnames := []string{}
	foundNames := make(map[string]struct{})
	processedSrc := re.ReplaceAllStringFunc(src, func(match string) string {
		// Get name of file to be included.
		ifname := re.FindStringSubmatch(match)[1]
		if ifname == fname {
			fileNotFoundError.files = append(fileNotFoundError.files, "Cannot include self.")
			return ""
		}

		if _, ok := foundNames[ifname]; ok {
			return ""
		} else {
			foundNames[ifname] = struct{}{}
		}

		// Add name of file to include files list.
		ifnames = append(ifnames, ifname)

		// Search incPaths for the file.
		// If not found, add this file to error list.
		ifpath, found := util.SearchPaths(ifname, incPaths)
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
		processedIncludeSrc, _, processedIncludeLocmap, err := preprocess(includeSrc, ifname, incPaths)
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
		return processedSrc, ifnames, locmap, fileNotFoundError
	}
	return processedSrc, ifnames, locmap, nil
}
