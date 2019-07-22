package syntax

import (
	"path"
	"testing"
)

// Tests that compilation fails when a file includes itself.
func TestFailSelfInclude(t *testing.T) {
	t.Parallel()
	if _, _, _, err := Compile(path.Join("testdata", "self_include.mro"),
		[]string{"testdata"}, false); err == nil {
		t.Error("expected an error.")
	}
}

// Tests that compilation fails when a file is in the transitive closure of its
// own includes.
func TestFailIncludeCycle(t *testing.T) {
	t.Parallel()
	if _, _, _, err := Compile(path.Join("testdata", "include_cycle_1.mro"),
		[]string{"testdata"}, false); err == nil {
		t.Error("expected an error.")
	}
	if _, _, _, err := Compile(path.Join("testdata", "include_cycle_2.mro"),
		[]string{"testdata"}, false); err == nil {
		t.Error("expected an error.")
	}
	if _, _, _, err := Compile(path.Join("testdata", "include_cycle_3.mro"),
		[]string{"testdata"}, false); err == nil {
		t.Error("expected an error.")
	}
}

// Tests that 1 including 2 and 3, both of which include 4, is legal.
func TestIncludeDiamond(t *testing.T) {
	t.Parallel()
	if _, ifnames, _, err := Compile(path.Join("testdata", "include_diamond_1.mro"),
		[]string{"testdata"}, false); err != nil {
		t.Error(err)
	} else {
		if len(ifnames) != 4 {
			t.Errorf("Expected 3 includes, found %d\n%v", len(ifnames), ifnames)
		}
		found := false
		for _, f := range ifnames {
			if f == "include_diamond_2.mro" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find pipeline.mro.")
		}
		found = false
		for _, f := range ifnames {
			if f == "include_diamond_3.mro" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find stages.mro.")
		}
		found = false
		for _, f := range ifnames {
			if f == "include_diamond_4.mro" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find stages.mro.")
		}
	}
}

// Tests that the "combined" source resulting from compilation is as expected.
func TestCombineSource(t *testing.T) {
	t.Parallel()
	if src, ifnames, _, err := Compile(path.Join("testdata", "call.mro"),
		[]string{"testdata"}, false); err != nil {
		t.Error(err)
	} else {
		if len(ifnames) != 2 {
			t.Errorf("Expected 3 included files, found %d", len(ifnames))
		}
		found := false
		for _, f := range ifnames {
			if f == "pipeline.mro" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find pipeline.mro.")
		}
		found = false
		for _, f := range ifnames {
			if f == "stages.mro" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find stages.mro.")
		}
		if src != `#
# @include "stages.mro"
#

# This tests combining of included sources.

filetype bam;

stage MY_STAGE(
    in  int info,
    out bam result,
    src py  "nope.py",
)

#
# @include "pipeline.mro"
#

pipeline MY_PIPELINE(
    in  int info,
    out bam result,
)
{
    call MY_STAGE(
        info = self.info,
    )

    return (
        result = MY_STAGE.result,
    )
}

# trailing comment
#
# @include "call.mro"
#

call MY_PIPELINE(
    info = 2,
)
` {
			t.Errorf("Incorrect combined source.  Got \n%s", src)
		}
	}
}

// Tests that FixIncludes does the right thing on a file with a top-level call.
func TestFixIncludesTop(t *testing.T) {
	t.Parallel()
	if src, err := FormatFile(path.Join("testdata", "call.mro"),
		true,
		[]string{"testdata"}); err != nil {
		t.Error(err)
	} else {
		if src != `# This file contains the top-level call.

@include "pipeline.mro"

call MY_PIPELINE(
    info = 2,
)
` {
			t.Errorf("Incorrect combined source.  Got \n%s", src)
		}
	}
}

// Tests that FixIncludes does the right thing on a file with a top-level call
// missing the include.
func TestFixIncludesTopMissing(t *testing.T) {
	t.Parallel()
	if src, err := FormatFile(path.Join("testdata", "call_missing_include.mro"),
		true,
		[]string{"testdata"}); err != nil {
		t.Error(err)
	} else {
		if src != `# This file contains the top-level call.

@include "pipeline.mro"

call MY_PIPELINE(
    info = 2,
)
` {
			t.Errorf("Incorrect combined source.  Got \n%s", src)
		}
	}
}

// Tests that FixIncludes does the right thing on a file with a pipeline.
func TestFixIncludesPipeline(t *testing.T) {
	t.Parallel()
	if src, err := FormatFile(path.Join("testdata", "pipeline.mro"),
		true,
		[]string{"testdata"}); err != nil {
		t.Error(err)
	} else {
		if src != `@include "stages.mro"

pipeline MY_PIPELINE(
    in  int info,
    out bam result,
)
{
    call MY_STAGE(
        info = self.info,
    )

    return (
        result = MY_STAGE.result,
    )
}
# trailing comment
` {
			t.Errorf("Incorrect combined source.  Got \n%s", src)
		}
	}
}

// Tests that FixIncludes does the right thing on a file with a pipeline
// with a missing include.
func TestFixIncludesPipelineMissing(t *testing.T) {
	t.Parallel()
	if src, err := FormatFile(path.Join("testdata", "pipeline_missing_include.mro"),
		true,
		[]string{"testdata"}); err != nil {
		t.Error(err)
	} else {
		if src != `@include "stages.mro"

pipeline MY_BROKEN_PIPELINE(
    in  int info,
    out bam result,
)
{
    call MY_STAGE(
        info = self.info,
    )

    return (
        result = MY_STAGE.result,
    )
}
# trailing comment
` {
			t.Errorf("Incorrect combined source.  Got \n%s", src)
		}
	}
}

// Tests that FixIncludes does the right thing on a file with a stage.
func TestFixIncludesStage(t *testing.T) {
	t.Parallel()
	if src, err := FormatFile(path.Join("testdata", "stages.mro"),
		true,
		[]string{"testdata"}); err != nil {
		t.Error(err)
	} else {
		if src != `# This tests combining of included sources.

filetype bam;

stage MY_STAGE(
    in  int info,
    out bam result,
    src py  "nope.py",
)
` {
			t.Errorf("Incorrect combined source.  Got \n%s", src)
		}
	}
}

// Tests that FixIncludes does the right thing on a file with a stage
// that is missing a type definition.
func TestFixIncludesStageMissing(t *testing.T) {
	t.Parallel()
	if src, err := FormatFile(path.Join("testdata", "stages_missing_type.mro"),
		true,
		[]string{"testdata"}); err != nil {
		t.Error(err)
	} else {
		if src != `filetype bam;

# This tests mrf --includes fixing type definitions.

stage MY_BROKEN_STAGE(
    in  int info,
    out bam result,
    src py  "nope.py",
)
` {
			t.Errorf("Incorrect combined source.  Got \n%s", src)
		}
	}
}

// Tests that compilation fails when a file includes itself.
func TestIncludeRelative(t *testing.T) {
	t.Parallel()
	if _, _, _, err := Compile(path.Join("testdata", "subdir", "pipeline_subdir.mro"),
		[]string{"testdata"}, false); err != nil {
		t.Error(err)
	}
	if _, _, _, err := Compile(path.Join("testdata", "subdir", "pipeline_rel.mro"),
		[]string{"testdata"}, false); err != nil {
		t.Error(err)
	}
}

// Tests that compilation fails when a file includes itself.
func TestFailAmbiguousInclude(t *testing.T) {
	t.Parallel()
	if _, _, _, err := Compile(path.Join("testdata", "subdir", "pipeline_ambiguous.mro"),
		[]string{"testdata"}, false); err == nil {
		t.Error("expected failure")
	}
}
