//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian formatter tests.
//

package syntax

import (
	"strings"
	"testing"
)

const fmtTestSrc = `# A super-simple test pipeline with forks.

# I am good at documenting my code with useful headers.

# Get my other stuff.
@include "my_special_stuff.mro"

# Files storing json.
filetype json;
filetype txt;

struct POINT(
    # x coordinate
    float x,
    # y coordinate
    int[] y,
)

# A thing with coordinates.
struct POINTALISM(
    POINT point,
    # output file
    json  dest  "help" "dest.xml",
)

struct HELPFUL(
    int  d2 "foo",
    file d3 ""    "blargh",
)

# Adds a key to the json in a file.
stage ADD_KEY1(
    # The key to add
    in  string key,
    # The value to add for this key.
    in  string value,
    # The file to read the initial dictionary from.
    in  json   start,
    # A file to check.  If the file exists, parse its content as a signal
    # for the job to send to itself.
    in  string failfile,
    # The output file.
    out json   result,
    # The source file.
    src py     "stages/add_key",
)

# Some more explanation of what I'm doing could go here.

# Adds a second key to the json in a file.
stage ADD_KEY2(
    in  string key       "The key to add",
    in  string value     "The value to set the key to",
    in  json   start,
    in  string failfile  "The file to check to force failure.",
    out json   result,
    out int    very_long_output_name_should_not_push_help_text_over,
    src py     "stages/add_key",
) using (
    special = "something",
)

# Adds a third key to the json in a file.
stage ADD_KEY3(
    in  string key,
    in  string value,
    in  json   start,
    in  string failfile,
    out json   result,
    out bool   disable_example,
    src py     "stages/add_key",
) retain (
    result,
)

stage ADD_KEY5(
    in  string   key,
    in  string[] value,
    src exec     "stages/whatever arg",
)

stage SUM_SQUARES(
    in  float[] values   "The values to square and then sum.",
    out float   sum,
    # default
    out int,
    src comp    "bin/sum_squares mode_arg",
) split (
    in  float   value,
    out float   square,
) using (
    # For some reason this uses lots of memory.
    mem_gb   = 4,
    # This doesn't generate files anyway.
    volatile = strict,
)

# Takes two files containing json dictionaries and merges them.
stage MERGE_JSON(
    in  json json1,
    in  json json2,
    out json result,
    src py   "stages/merge_json",
)

stage MERGE_JSON2(
    in  json[] input,
    src py     "stages/merge_json",
)

stage MAP_EXAMPLE(
    in  map foo,
    src py  "stages/merge_json",
) using (
    mem_gb   = 2,
    # This stage always uses 4 threads!
    threads  = 4,
    # This stage uses 2TB of vmem.
    vmem_gb  = 1024,
    volatile = strict,
)

# This stage takes a struct input
stage STRUCT_CONSUMER(
    in  POINTALISM foo,
    in  int[]      y,
    in  float[]    xarr,
    in  POINT      point,
    # x coordinate
    out float      x,
    # y coordinate
    out int[]      y,
    out POINT      point,
    # output file
    out json       dest   "outname.json",
    out POINTALISM extra,
    out POINT[]    arr,
    src comp       "stages/structy",
)

stage MAP_PRODUCER(
    in  STRUCT_CONSUMER          input,
    out map<STRUCT_CONSUMER>     flat_map,
    out map<STRUCT_CONSUMER>[]   array_of_map,
    out map<STRUCT_CONSUMER[]>   map_of_array,
    out map<STRUCT_CONSUMER[]>[] array_of_map_of_array,
    out STRUCT_CONSUMER[]        array,
    src comp                     "stages/structy",
)

stage MAP_CONSUMER(
    in  map<STRUCT_CONSUMER> foo1,
    in  map<STRUCT_CONSUMER> foo2,
    in  POINT[]              point_array,
    in  map<int[]>           map_of_array,
    in  int[][]              array_2d,
    in  map<float[]>         map_of_float_array,
    in  map<float>[]         array_of_float_map,
    src comp                 "stages/structy",
)

#
stage _HAS_DEFAULT_OUT(
    out path,
    src comp "fake",
)

pipeline USES_DEFAULT(
    out path something,
)
{
    call _HAS_DEFAULT_OUT(
    )

    return (
        something = _HAS_DEFAULT_OUT.default,
    )
}

# Adds some keys to some json files and then merges them.
pipeline AWESOME(
    in  string     key1       "help text",
    in  string     value1,
    in  string     key2,
    in  string     value2,
    in  POINTALISM struct_in,
    out json       outfile    "The json file containing all of the keys and values."  "all_keys",
)
{
    call ADD_KEY1(
        key      = self.key1,
        value    = self.value1,
        failfile = "fail \n\"1\"",
        start    = null,
    ) using (
        local = true,
    )

    call ADD_KEY2(
        key      = self.key2,
        value    = self.value2,
        failfile = "fail2",
        start    = ADD_KEY1.result,
    )

    call ADD_KEY3(
        key      = "3",
        value    = "three",
        failfile = "fail3",
        start    = ADD_KEY2.result,
    )

    call ADD_KEY1 as ADD_KEY4(
        key      = "4",
        value    = sweep(
            "four",
            "feir",
        ),
        failfile = "fail4",
        start    = ADD_KEY2.result,
    )

    call MAP_EXAMPLE(
        foo = {
            "bar": "baz",
            "bing": null,
            "blarg": {
                "n": 2,
            },
        },
    ) using (
        # ADD_KEY3 can disable this stage.
        disabled = ADD_KEY3.disable_example,
        local    = true,
        # This shouldn't be volatile because reasons.
        volatile = false,
    )

    call MAP_EXAMPLE as MAP_EXAMPLE2(
        foo = {},
    )

    call ADD_KEY5(
        key   = "5",
        value = ["five"],
    ) using (
        volatile = true,
    )

    call ADD_KEY5 as ADD_KEY6(
        key   = "6",
        value = [
            "six",
            "seven",
        ],
    )

    call MERGE_JSON(
        json1 = ADD_KEY3.result,
        json2 = ADD_KEY4.result,
    )

    call MERGE_JSON2(
        input = [ADD_KEY3.result],
    )

    call MERGE_JSON2 as MERGE_JSON3(
        input = [
            ADD_KEY3.result,
            ADD_KEY4.result,
        ],
    )

    call MERGE_JSON2 as MERGE_JSON4(
        input = [
            "four",
            ADD_KEY4.result,
        ],
    )

    call MERGE_JSON2 as MERGE_JSON5(
        input = [],
    )

    call STRUCT_CONSUMER(
        foo   = self.struct_in,
        y     = self.struct_in.point.y,
        xarr  = null,
        point = self.struct_in.point,
    )

    call STRUCT_CONSUMER as CANIBAL_1(
        foo   = STRUCT_CONSUMER,
        y     = STRUCT_CONSUMER.y,
        xarr  = STRUCT_CONSUMER.arr.x,
        point = STRUCT_CONSUMER.point,
    )

    call STRUCT_CONSUMER as CANIBAL_2(
        foo   = STRUCT_CONSUMER.extra,
        y     = STRUCT_CONSUMER.extra.point.y,
        xarr  = STRUCT_CONSUMER.arr.x,
        point = STRUCT_CONSUMER.extra.point,
    )

    call MAP_PRODUCER(
        input = STRUCT_CONSUMER,
    )

    call MAP_CONSUMER(
        foo1               = {
            "item": STRUCT_CONSUMER,
        },
        foo2               = MAP_PRODUCER.flat_map,
        point_array        = MAP_PRODUCER.array.point,
        map_of_array       = MAP_PRODUCER.flat_map.y,
        array_2d           = MAP_PRODUCER.array.y,
        map_of_float_array = MAP_PRODUCER.map_of_array.x,
        array_of_float_map = MAP_PRODUCER.array_of_map.x,
    )

    return (
        outfile = MERGE_JSON.result,
    )

    retain (
        ADD_KEY1.result,
    )
}

# Calls the pipelines, sweeping over two forks.
call AWESOME(
    key1      = "1",
    value1    = "one",
    key2      = "2",
    value2    = sweep(
        "two",
        "deux",
    ),
    struct_in = {
        dest: "foo.json",
        point: {
            x: 1.5,
            y: [
                2,
                3,
            ],
        },
    },
)
`

func TestFormatCommentedSrc(t *testing.T) {
	src := fmtTestSrc
	if formatted, err := Format(src, "test", false, nil); err != nil {
		t.Errorf("Format error: %v", err)
	} else if formatted != src {
		diffLines(src, formatted, t)
	}
}

func BenchmarkFormat(b *testing.B) {
	srcFile := new(SourceFile)
	if ast, err := yaccParse([]byte(fmtTestSrc),
		srcFile, makeStringIntern()); err != nil {
		b.Error(err)
	} else {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ast.format(false)
		}
	}
}

func TestFormatTopoSort(t *testing.T) {
	const src = `pipeline PIPELINE(
    in  int input,
    out int output1,
    out int output2,
)
{
    call STAGE_3(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    call STAGE as STAGE_1(
        input = self.input,
    )

    call STAGE as STAGE_2(
        input = self.input,
    )

    call STAGE as STAGE_4(
        input = STAGE_2.output,
    )

    call STAGE_3 as STAGE_5(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    return (
        output1 = STAGE_3.output,
        output2 = STAGE_5.output,
    )
}
`
	const expected = `pipeline PIPELINE(
    in  int input,
    out int output1,
    out int output2,
)
{
    call STAGE as STAGE_1(
        input = self.input,
    )

    call STAGE as STAGE_2(
        input = self.input,
    )

    call STAGE_3(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    call STAGE as STAGE_4(
        input = STAGE_2.output,
    )

    call STAGE_3 as STAGE_5(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    return (
        output1 = STAGE_3.output,
        output2 = STAGE_5.output,
    )
}
`
	if formatted, err := Format(src, "test", false, nil); err != nil {
		t.Errorf("Format error: %v", err)
	} else if formatted != expected {
		diffLines(expected, formatted, t)
	}
}

// Produce a relatively debuggable side-by-side diff.
func diffLines(src, formatted string, t *testing.T) {
	src_lines := strings.Split(src, "\n")
	formatted_lines := strings.Split(formatted, "\n")
	offset := 0
	// Replace tabs with \t for visibility, and truncate lines over 30
	// characters so that the diff will fit.
	trimLine := func(line string) string {
		line = strings.Replace(line, "\t", "\\t", -1)
		if len(line) > 30 {
			line = line[:27] + "..."
		}
		return line
	}
	removeSpace := func(line string) string {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			return line
		}
		flds := strings.Fields(line)
		if len(flds) <= 1 {
			return line
		}
		return strings.Join(flds, "")
	}
	for i, line := range src_lines {
		pad := ""
		if fmtLen := len(line) + strings.Count(line, "\t"); fmtLen < 30 {
			// Add the tab count because when formatting we'll replace those
			// characters with \t.
			pad = strings.Repeat(" ", 30-fmtLen)
		}
		if len(formatted_lines) > i+offset {
			if line == formatted_lines[i+offset] {
				line = trimLine(line)
				t.Logf("%3d: %s %s= %s", i, line, pad, line)
			} else if cline := removeSpace(line); cline == removeSpace(formatted_lines[i+offset]) {
				t.Errorf("%3d: %s %s| %s", i, line, pad, formatted_lines[i+offset])
			} else {
				forwardOffset := 0
				for moreOffset, fline := range formatted_lines[i+offset:] {
					if removeSpace(fline) == cline {
						forwardOffset = moreOffset
						break
					} else if moreOffset > 20 {
						break
					}
				}
				backwardOffset := 0
				cline := removeSpace(formatted_lines[i+offset])
				for moreOffset, uline := range src_lines[i:] {
					if removeSpace(uline) == cline {
						backwardOffset = moreOffset
						break
					} else if moreOffset > 20 {
						break
					}
				}
				if forwardOffset == 0 && backwardOffset == 0 {
					t.Errorf("%3d: %s %s| %s", i, line, pad, formatted_lines[i+offset])
				} else if (forwardOffset == 0 && backwardOffset != 0) ||
					(backwardOffset > forwardOffset) {
					t.Errorf("%3d: %s %s<", i, line, pad)
					offset--
				} else {
					for j := 0; j < forwardOffset; j++ {
						t.Errorf("%s > %s", strings.Repeat(" ", 35), formatted_lines[i+j+offset])
					}
					offset += forwardOffset
					if line == formatted_lines[i+offset] {
						line = trimLine(line)
						t.Logf("%3d: %s %s= %s", i, line, pad, line)
					} else {
						t.Errorf("%3d: %s %s| %s", i, line, pad, formatted_lines[i+offset])
					}
				}
			}
		} else {
			t.Errorf("%3d: %s %s<", i, line, pad)
		}
	}
	if len(formatted_lines) > len(src_lines)+offset {
		for _, line := range formatted_lines[len(src_lines)+offset:] {
			t.Errorf("%s > %s", strings.Repeat(" ", 35), line)
		}
	}
}
