// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

package core

import (
	"encoding/json"
	"testing"
)

const filenameTestOuts = `{
    "read_groups": [
        "74484:MissingLibrary:1:HFJK5DMXX:2", 
        "74484:MissingLibrary:1:HFJK5DMXX:2", 
        "74484:MissingLibrary:1:HFJK5DMXX:2" 
    ], 
    "bam_comments": [
        "bam_to_fastq:I1(BC:QT)", 
        "bam_to_fastq:R1(CR:CY,UR:UY)", 
        "bam_to_fastq:R2(SEQ:QUAL)"
    ], 
    "gem_groups": [
        1, 
        1, 
        1
    ], 
    "barcode_counts": "/path/to/stage/fork0/join-ub5d43fd603/files/barcode_counts.json", 
    "tags": [
        "/path/to/stage/fork0/chnk132-ub5d43fd606/files/tags.fastq.lz4", 
        "/path/to/stage/fork0/chnk133-ub5d43fd606/files/tags.fastq.lz4", 
        "/path/to/stage/fork0/chnk134-ub5d43fd606/files/tags.fastq.lz4"
    ], 
    "align": {
        "aligner": "star", 
        "high_conf_mapq": 255
    }, 
    "feature_counts": "/path/to/stage/fork0/join-ub5d43fd603/files/feature_counts.json", 
    "read2s": [
        "/path/to/stage/fork0/chnk132-ub5d43fd606/files/read2s.fastq", 
        "/path/to/stage/fork0/chnk133-ub5d43fd606/files/read2s.fastq", 
        "/path/to/stage/fork0/chnk134-ub5d43fd606/files/read2s.fastq"
    ],
    "something": {
        "stuff": "/path/to/file.hdf5"
    },
    "library_types": [
        "Gene Expression", 
        "Gene Expression", 
        "Gene Expression"
    ], 
    "summary": "/path/to/stage/fork0/join-ub5d43fd603/files/summary.json", 
    "read1s": [
        "/path/to/stage/fork0/chnk132-ub5d43fd606/files/read1s.fastq.lz4", 
        "/path/to/stage/fork0/chnk133-ub5d43fd606/files/read1s.fastq.lz4", 
        "/path/to/stage/fork0/chnk134-ub5d43fd606/files/read1s.fastq.lz4"
    ]
}`

// Tests that getMaybeFileNames correctly finds file names in json.
func TestGetMaybeFileNames(t *testing.T) {
	var outs LazyArgumentMap
	if err := json.Unmarshal([]byte(filenameTestOuts), &outs); err != nil {
		t.Fatal(err)
	}
	for _, arg := range []string{
		"read_groups",
		"bam_comments",
		"gem_groups",
		"align",
		"library_types",
	} {
		if out := outs[arg]; len(out) < 2 {
			t.Errorf("Expected output arg %s missing", arg)
		} else if files := getMaybeFileNames(out); len(files) > 0 {
			t.Errorf("Expected no file names in %s, got %d (first = %s)",
				arg, len(files), files[0])
		}
	}
	for _, arg := range []string{
		"tags",
		"read2s",
		"read1s",
	} {
		if out := outs[arg]; len(out) < 2 {
			t.Errorf("Expected output arg %s missing", arg)
		} else if files := getMaybeFileNames(out); len(files) != 3 {
			t.Errorf("Expected 3 file names in %s, got %d from %s",
				arg, len(files), string(out))
		}
	}
	for _, arg := range []string{
		"barcode_counts",
		"feature_counts",
		"summary",
		"something",
	} {
		if out := outs[arg]; len(out) < 2 {
			t.Errorf("Expected output arg %s missing", arg)
		} else if files := getMaybeFileNames(out); len(files) != 1 {
			t.Errorf("Expected 1 file name in %s, got %d from %s",
				arg, len(files), string(out))
		}
	}
}
