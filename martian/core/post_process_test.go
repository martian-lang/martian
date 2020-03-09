package core

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/martian-lang/martian/martian/util"
)

type fooStruct struct {
	File1 string `json:"file1"`
}
type creatorStage struct {
	Bar   *int   `json:"bar"`
	File1 string `json:"file1"`
	File2 string `json:"file2"`
	File3 string `json:"file3"`
}

type innerPipeline struct {
	Bar      creatorStage             `json:"bar"`
	Results1 map[string]*creatorStage `json:"results1"`
	Results2 map[string]*creatorStage `json:"results2"`
}

type outerPipeline struct {
	Text   string                   `json:"text"`
	Inner  innerPipeline            `json:"inner"`
	Files1 []map[string]string      `json:"files1"`
	Bars   []int                    `json:"bars"`
	Strs   []fooStruct              `json:"strs"`
	Texts  []string                 `json:"texts"`
	One    *creatorStage            `json:"one"`
	Many   map[string]*creatorStage `json:"many"`
}

func compareOutputText(t *testing.T, expected, actual string) {
	t.Helper()
	if str := strings.TrimSpace(actual); str != expected {
		t.Error(str)
		expectLines := strings.Split(expected, "\n")
		actualLines := strings.Split(str, "\n")
		if len(expectLines) != len(actualLines) {
			t.Errorf("line count %d != %d", len(actualLines), len(expectLines))
		} else {
			for i, exp := range expectLines {
				if exp != actualLines[i] {
					te := strings.TrimSpace(exp)
					ta := strings.TrimSpace(actualLines[i])
					if te == ta {
						if strings.TrimLeftFunc(
							exp, unicode.IsSpace) == strings.TrimLeftFunc(
							actualLines[i], unicode.IsSpace) {
							t.Errorf("line %d: leading whitespace differences", i)
						} else if strings.TrimRightFunc(
							exp, unicode.IsSpace) == strings.TrimRightFunc(
							actualLines[i], unicode.IsSpace) {
							t.Errorf("line %d: trailing whitespace differences", i)
						} else {
							t.Errorf("line %d: whitespace differences", i)
						}
					} else {
						t.Errorf("line %d:\t%s != %s", i, ta, te)
					}
				}
			}
		}
	}
}

func TestPostProcess(t *testing.T) {
	util.MockSignalHandlersForTest()
	psOuts, err := filepath.Abs("testdata/test_post_process_struct_pipestance/outs")
	if err != nil {
		t.Fatal(err)
	}
	psPath := filepath.Dir(psOuts)
	psSrc := filepath.Join(psPath, "srcs")
	if err := os.RemoveAll(psPath); err != nil &&
		!os.IsNotExist(err) {
		t.Error(err)
	}
	if err := os.MkdirAll(psSrc, 0755); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if !t.Failed() {
			os.RemoveAll(psPath)
		}
	}()

	src, err := ioutil.ReadFile("testdata/struct_pipeline.mro")
	if err != nil {
		t.Fatal(err)
	}

	// Set up minimal runtime without actually loading config.
	conf := DefaultRuntimeOptions()
	rt := Runtime{
		Config: &conf,
		LocalJobManager: &LocalJobManager{
			jobSettings: new(JobManagerSettings),
		},
	}
	rt.JobManager = rt.LocalJobManager
	_, _, pipestance, err := rt.instantiatePipeline(string(src),
		"testdata/struct_pipeline.mro",
		"test_struct_pipeline", psPath, nil,
		"none", nil, false, false, context.Background())
	if err != nil {
		os.RemoveAll(psPath)
		t.Fatal(err)
	}
	files := make([]string, 13)
	for i := range files {
		files[i] = filepath.Join(psSrc, fmt.Sprintf("file%d.txt", i))
	}
	for i, f := range files {
		if err := ioutil.WriteFile(f, []byte(strconv.Itoa(i)), 0644); err != nil {
			t.Error(err)
		}
	}
	fork := pipestance.node.forks[0]

	b := 1
	c1 := creatorStage{
		Bar:   &b,
		File1: files[1],
		File2: files[2],
		File3: files[3],
	}
	c2 := creatorStage{
		File1: files[4],
		File2: files[5],
	}
	c3 := creatorStage{
		File1: files[6],
	}
	c4 := creatorStage{
		File3: files[7],
	}
	r1 := map[string]*creatorStage{
		"c1": &c2,
		"c2": &c3,
	}
	if err := fork.metadata.Write(OutsFile, &outerPipeline{
		Text: files[0],
		Inner: innerPipeline{
			Bar:      c1,
			Results1: r1,
			Results2: map[string]*creatorStage{
				"c1": &c4,
				"c2": nil,
			},
		},
		Files1: []map[string]string{
			{
				"c1": files[8],
				"c2": files[9],
			},
			{
				"c1": files[10],
				"c2": files[11],
			},
		},
		Bars: []int{2, 3},
		Strs: []fooStruct{
			{File1: "foo1"},
			{File1: "foo2"},
		},
		Texts: files[12:],
		One:   &c3,
		Many:  r1,
	}); err != nil {
		t.Error(err)
	}
	var buf strings.Builder
	util.SetPrintLogger(&buf)
	if err := fork.postProcess(); err != nil {
		t.Error(err)
	}
	util.SetPrintLogger(&devNull)
	for _, f := range files {
		if info, err := os.Lstat(f); err != nil {
			t.Error(err)
		} else if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("expected symlink for %s", f)
		}
	}
	check := func(fn, expect string) {
		t.Helper()
		if b, err := ioutil.ReadFile(filepath.Join(psOuts, fn)); err != nil {
			t.Error(err)
		} else if s := string(b); s != expect {
			t.Errorf("expected %s in %s, got %s",
				expect, fn, s)
		}
	}
	check("text.txt", "0")
	check("inner/bar/file1.txt", "1")
	check("inner/bar/file2", "2")
	check("inner/bar/output_name.file", "3")
	check("inner/results1/c1/file1.txt", "4")
	check("inner/results1/c1/file2", "5")
	check("inner/results1/c2/file1.txt", "6")
	check("inner/output_name/c1/output_name.file", "7")
	check("files1/0/c1.txt", "8")
	check("files1/0/c2.txt", "9")
	check("files1/1/c1.txt", "10")
	check("files1/1/c2.txt", "11")
	if _, err := os.Stat(filepath.Join(psOuts, "inner/output_name/c2")); err == nil {
		t.Error("unexpected file inner/output_name/c2")
	}
	const expectSummary = `Outputs:
- one text file: outs/text.txt
- inner:
    bar:
      bar:       1
      file1:     outs/inner/bar/file1.txt
      file2:     outs/inner/bar/file2
      help text: outs/inner/bar/output_name.file
    results1:    {
      c1:
        bar:       null
        file1:     outs/inner/results1/c1/file1.txt
        file2:     outs/inner/results1/c1/file2
        help text: null
      c2:
        bar:       null
        file1:     outs/inner/results1/c2/file1.txt
        file2:     null
        help text: null
    }
    description: {
      c1:
        bar:       null
        file1:     null
        file2:     null
        help text: outs/inner/output_name/c1/output_name.file
      c2: null
    }
- files1:        [
    0: {
      c1: outs/files1/0/c1.txt
      c2: outs/files1/0/c2.txt
    }
    1: {
      c1: outs/files1/1/c1.txt
      c2: outs/files1/1/c2.txt
    }
  ]
- some ints:     [2,3]
- strs:          [
    {"file1":"foo1"}
    {"file1":"foo2"}
  ]
- some files:    [
    0: outs/output_text_file_set/0.txt
  ]
- one:
    bar:       null
    file1:     outs/inner/results1/c2/file1.txt
    file2:     null
    help text: null
- many files:    {
    c1:
      bar:       null
      file1:     outs/inner/results1/c1/file1.txt
      file2:     outs/inner/results1/c1/file2
      help text: null
    c2:
      bar:       null
      file1:     outs/inner/results1/c2/file1.txt
      file2:     null
      help text: null
  }`
	compareOutputText(t, expectSummary,
		strings.Replace(buf.String(), psPath+"/", "", -1))
	var outs outerPipeline
	if err := fork.metadata.ReadInto(OutsFile, &outs); err != nil {
		t.Error(err)
	} else {
		if s := strings.TrimPrefix(outs.Text, psPath); s != "/outs/text.txt" {
			t.Errorf("%s != /outs/text.txt", s)
		}
		if outs.Inner.Bar.Bar == nil {
			t.Error("inner.bar.bar was null")
		} else if v := *outs.Inner.Bar.Bar; v != 1 {
			t.Errorf("inner.bar.bar == %d != 1", v)
		}
		if s := strings.TrimPrefix(outs.Inner.Bar.File1, psPath); s != "/outs/inner/bar/file1.txt" {
			t.Errorf("inner.bar.file1 == '%s' != '/outs/inner/bar/file1.txt'", s)
		}
		if s := strings.TrimPrefix(outs.Inner.Bar.File2, psPath); s != "/outs/inner/bar/file2" {
			t.Errorf("inner.bar.file2 == '%s' != '/outs/inner/bar/file2'", s)
		}
		if s := strings.TrimPrefix(outs.Inner.Bar.File3, psPath); s != "/outs/inner/bar/output_name.file" {
			t.Errorf("inner.bar.file3 == '%s' != '/outs/inner/bar/output_name.file'", s)
		}
		if d := len(outs.Inner.Results1); d != 2 {
			t.Errorf("len(inner.results1) == %d != 2", d)
		}
		if d := len(outs.Inner.Results2); d != 2 {
			t.Errorf("len(inner.results2) == %d != 2", d)
		}
		if len(outs.Bars) != 2 {
			t.Errorf("len(bars) == %d != 2", len(outs.Bars))
		}
		if len(outs.Strs) != 2 {
			t.Errorf("len(strs) == %d != 2", len(outs.Strs))
		}
		if len(outs.Texts) != 1 {
			t.Errorf("len(texts) == %d != 2", len(outs.Texts))
		} else if s := strings.TrimPrefix(outs.Texts[0], psPath); s != `/outs/output_text_file_set/0.txt` {
			t.Errorf("texts[0] == '%s' != '/outs/output_text_file_set/0.txt'", s)
		}
	}
}

func TestPostProcessEmpties(t *testing.T) {
	util.MockSignalHandlersForTest()
	psOuts, err := filepath.Abs("testdata/test_post_process_empties_struct_pipestance/outs")
	if err != nil {
		t.Fatal(err)
	}
	psPath := filepath.Dir(psOuts)
	psSrc := filepath.Join(psPath, "srcs")
	if err := os.RemoveAll(psPath); err != nil &&
		!os.IsNotExist(err) {
		t.Error(err)
	}
	if err := os.MkdirAll(psSrc, 0755); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if !t.Failed() {
			os.RemoveAll(psPath)
		}
	}()

	src, err := ioutil.ReadFile("testdata/struct_pipeline.mro")
	if err != nil {
		t.Fatal(err)
	}

	// Set up minimal runtime without actually loading config.
	conf := DefaultRuntimeOptions()
	rt := Runtime{
		Config: &conf,
		LocalJobManager: &LocalJobManager{
			jobSettings: new(JobManagerSettings),
		},
	}
	rt.JobManager = rt.LocalJobManager
	_, _, pipestance, err := rt.instantiatePipeline(string(src),
		"testdata/struct_pipeline.mro",
		"test_struct_pipeline", psPath, nil,
		"none", nil, false, false, context.Background())
	if err != nil {
		os.RemoveAll(psPath)
		t.Fatal(err)
	}
	files := make([]string, 3)
	for i := range files {
		files[i] = filepath.Join(psSrc, fmt.Sprintf("file%d.txt", i))
	}
	for i, f := range files {
		if err := ioutil.WriteFile(f, []byte(strconv.Itoa(i)), 0644); err != nil {
			t.Error(err)
		}
	}
	// Use the executable file as a path that's known to exist and that is
	// outside the pipestance directory.
	var exeFile, exeLink string
	if exe, err := os.Executable(); err != nil {
		t.Error(err)
	} else {
		exeFile = exe
		exeLink = filepath.Join(psSrc, fmt.Sprintf("file%d.txt", len(files)))
		files = append(files, exeLink)
		if err := os.Symlink(exe, exeLink); err != nil {
			t.Error(err)
		}
	}
	fork := pipestance.node.forks[0]

	b := 1
	c1 := creatorStage{
		Bar:   &b,
		File1: files[0],
		File2: files[1],
		File3: files[2],
	}
	c2 := creatorStage{
		File1: files[0],
		File2: files[1],
	}
	c3 := creatorStage{
		File1: exeFile,
	}
	if err := fork.metadata.Write(OutsFile, &outerPipeline{
		Text: exeLink,
		Inner: innerPipeline{
			Bar: c1,
			Results1: map[string]*creatorStage{
				"c1": &c2,
				"c2": &c3,
			},
			Results2: map[string]*creatorStage{},
		},
		Files1: []map[string]string{},
		Strs:   []fooStruct{},
	}); err != nil {
		t.Error(err)
	}
	var buf strings.Builder
	util.SetPrintLogger(&buf)
	if err := fork.postProcess(); err != nil {
		t.Error(err)
	}
	util.SetPrintLogger(&devNull)
	for _, f := range files {
		if info, err := os.Lstat(f); err != nil {
			t.Error(err)
		} else if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("expected symlink for %s", f)
		}
	}
	if dest, err := os.Readlink(filepath.Join(psOuts, "text.txt")); err != nil {
		t.Error("reading link in outs: ", err)
	} else if dest != exeFile {
		t.Errorf("outs link points to wrong location: %s != %s", dest, exeFile)
	}
	if dest, err := os.Readlink(filepath.Join(psOuts, "inner/results1/c2/file1.txt")); err != nil {
		t.Error("reading link in outs: ", err)
	} else if dest != exeFile {
		t.Errorf("outs link points to wrong location: %s != %s", dest, exeFile)
	}
	check := func(fn, expect string) {
		t.Helper()
		if b, err := ioutil.ReadFile(filepath.Join(psOuts, fn)); err != nil {
			t.Error(err)
		} else if s := string(b); s != expect {
			t.Errorf("expected %s in %s, got %s",
				expect, fn, s)
		}
	}
	check("inner/bar/file1.txt", "0")
	check("inner/bar/file2", "1")
	check("inner/bar/output_name.file", "2")
	check("inner/results1/c1/file1.txt", "0")
	check("inner/results1/c1/file2", "1")
	if _, err := os.Stat(filepath.Join(psOuts, "inner/output_name/c2")); err == nil {
		t.Error("unexpected file inner/output_name/c2")
	}
	expectSummary := fmt.Sprintf(`Outputs:
- one text file: %s
- inner:
    bar:
      bar:       1
      file1:     outs/inner/bar/file1.txt
      file2:     outs/inner/bar/file2
      help text: outs/inner/bar/output_name.file
    results1:    {
      c1:
        bar:       null
        file1:     outs/inner/bar/file1.txt
        file2:     outs/inner/bar/file2
        help text: null
      c2:
        bar:       null
        file1:     %s
        file2:     null
        help text: null
    }
    description: {}
- files1:        []
- some ints:     null
- strs:          []
- some files:    null
- one:           null
- many files:    null`, exeFile, exeFile)
	compareOutputText(t, expectSummary,
		strings.Replace(buf.String(), psPath+"/", "", -1))
}
