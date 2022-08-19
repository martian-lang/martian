package core

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"testing"
)

func TestPathIsInside(t *testing.T) {
	if !pathIsInside("/path/to/thing/", "/path/to/thing") {
		t.Error("/path/to/thing should be inside itself")
	}
	if !pathIsInside("/path/to/thing", "/path/to") {
		t.Error("/path/to/thing should be inside /path/to")
	}
	if !pathIsInside("/path/to/thing", "/path/") {
		t.Error("/path/to/thing should be inside /path/")
	}
	if pathIsInside("/path/to/thing", "/not/path/to/thing") {
		t.Error("/path/to/thing should not be inside /not/path/to/thing")
	}
}

func TestGetArgsToFilesMap(t *testing.T) {
	t.Parallel()
	forkDir, err := os.MkdirTemp("", "testGetArgsToFilesMap")
	if err != nil {
		t.Skip(err)
	}
	defer os.RemoveAll(forkDir)

	fileArgs := map[string]map[Nodable]struct{}{
		"thing1":      nil,
		"thing2":      nil,
		"things":      nil,
		"what":        nil,
		"link":        nil,
		"otherThings": nil,
	}
	realPaths := []string{
		path.Join(forkDir, "thing", "thing1"),
		path.Join(forkDir, "thing", "thing2"),
		path.Join(forkDir, "thing", "thing3"),
		path.Join(forkDir, "what"),
	}
	link := path.Join(forkDir, "thing1link")
	for _, p := range realPaths {
		if err := os.MkdirAll(path.Dir(p), 0755); err != nil {
			t.Error(err)
		}
		if err := ioutil.WriteFile(p, []byte(p), 0644); err != nil {
			t.Error(err)
		}
	}
	if err := os.Symlink(realPaths[0], link); err != nil {
		t.Error(err)
	}

	var outs LazyArgumentMap
	{
		outsFrom := map[string]interface{}{
			"thing1":      realPaths[0],
			"thing2":      realPaths[1],
			"things":      path.Join(forkDir, "thing"),
			"otherThings": path.Join(forkDir, "not", "a", "path"),
			"what":        realPaths[3],
			"link":        link,
		}
		if b, err := json.Marshal(outsFrom); err != nil {
			t.Fatal(err)
		} else if err := json.Unmarshal(b, &outs); err != nil {
			t.Fatal(err)
		}
	}
	result := getArgsToFilesMap(fileArgs, outs, true, "test")
	if result == nil {
		t.Fatal("No result map")
	}
	if fs := result["thing1"]; fs == nil {
		t.Error("thing1 was nil.")
	} else if len(fs) != 1 {
		t.Errorf("Expected 1 file for thing1, got %d",
			len(fs))
	} else if _, ok := fs[realPaths[0]]; !ok {
		t.Errorf("thing1 did not keep %s alive.", realPaths[0])
	}
	if fs := result["thing2"]; fs == nil {
		t.Error("thing2 was nil.")
	} else if len(fs) != 1 {
		t.Errorf("Expected 1 file for thing2, got %d",
			len(fs))
	} else if _, ok := fs[realPaths[1]]; !ok {
		t.Errorf("thing2 did not keep %s alive.", realPaths[1])
	}
	if fs := result["things"]; fs == nil {
		t.Error("things was nil.")
	} else if len(fs) != 1 {
		t.Errorf("Expected 1 files for things, got %d",
			len(fs))
	} else {
		p := path.Join(forkDir, "thing")
		if _, ok := fs[p]; !ok {
			t.Errorf("things did not keep %s alive.", p)
		}
	}
	if fs := result["what"]; fs == nil {
		t.Error("what was nil.")
	} else if len(fs) != 1 {
		t.Errorf("Expected 1 file for what, got %d",
			len(fs))
	} else if _, ok := fs[realPaths[3]]; !ok {
		t.Errorf("what did not keep %s alive.", realPaths[3])
	}
	if fs := result["link"]; fs == nil {
		t.Error("link was nil.")
	} else if len(fs) != 2 {
		t.Errorf("Expected 1 file for link, got %d",
			len(fs))
	} else if _, ok := fs[realPaths[0]]; !ok {
		t.Errorf("link did not keep %s alive.", realPaths[0])
	} else if _, ok := fs[link]; !ok {
		t.Errorf("link did not keep %s alive.", link)
	}
	if _, ok := result["otherThings"]; ok {
		t.Error("otherThings was present.")
	}
}

func makeStorageTestDir(t testing.TB, forkDir string) (
	map[string]map[Nodable]struct{},
	[]string,
	LazyArgumentMap,
	string,
	[]string,
	error) {
	t.Helper()
	fileArgs := map[string]map[Nodable]struct{}{
		"thing1":      nil,
		"thing2":      nil,
		"things":      nil,
		"what":        nil,
		"link":        nil,
		"otherThings": nil,
		"dir":         nil,
	}
	realPaths := []string{
		path.Join(forkDir, "thing", "thing1"),
		path.Join(forkDir, "thing", "thing2"),
		path.Join(forkDir, "thing", "thing3"),
		path.Join(forkDir, "what"),
		path.Join(forkDir, "unreferenced"),
	}
	pathsList := make([]string, 200)
	for i := range pathsList {
		pathsList[i] = path.Join(forkDir, "dir", strconv.Itoa(i))
	}
	realPaths = append(realPaths, pathsList...)
	link := path.Join(forkDir, "thing1link")
	for _, p := range realPaths {
		if err := os.MkdirAll(path.Dir(p), 0755); err != nil {
			t.Error(err)
		}
		if err := ioutil.WriteFile(p, []byte(p), 0644); err != nil {
			t.Error(err)
		}
	}
	if err := os.Symlink(realPaths[0], link); err != nil {
		t.Error(err)
	}
	var outs LazyArgumentMap
	{
		outsFrom := map[string]interface{}{
			"thing1":      realPaths[0],
			"thing2":      realPaths[1],
			"things":      path.Join(forkDir, "thing"),
			"otherThings": path.Join(forkDir, "not", "a", "path"),
			"what":        realPaths[3],
			"link":        link,
			"dir":         pathsList,
		}
		if b, err := json.Marshal(outsFrom); err != nil {
			return nil, nil, nil, "", nil, err
		} else if err := json.Unmarshal(b, &outs); err != nil {
			return nil, nil, nil, "", nil, err
		}
	}
	return fileArgs, realPaths, outs, link, pathsList, nil
}

func TestAddFilesToArgsMappings(t *testing.T) {
	t.Parallel()
	forkDir, err := ioutil.TempDir("", "testAddFilesToArgsMappings")
	if err != nil {
		t.Skip(err)
	}
	defer os.RemoveAll(forkDir)
	fileArgs, realPaths, outs, link, pathsList, err := makeStorageTestDir(t, forkDir)
	if err != nil {
		t.Fatal(err)
	}
	argToFiles := getArgsToFilesMap(fileArgs, outs, true, "test")
	if argToFiles == nil {
		t.Fatal("No argToFiles map")
	}
	filesToArgs := make(map[string]*vdrFileCache, len(fileArgs))
	addFilesToArgsMappings(forkDir, true, "test",
		filesToArgs, argToFiles)
	checkFileToArgMappings(t, filesToArgs, realPaths, forkDir, link, pathsList)
}

type errorReporter interface {
	Errorf(format string, args ...interface{})
}

func checkFileToArgMappings(t errorReporter, filesToArgs map[string]*vdrFileCache,
	realPaths []string, forkDir, link string, pathsList []string) {
	if args := filesToArgs[realPaths[0]]; args == nil {
		t.Errorf("%s had no results.", realPaths[0])
	} else {
		if len(args.args) != 3 {
			t.Errorf("Expected 3 refs for %s, got %d",
				realPaths[0], len(args.args))
		}
		if _, ok := args.args["thing1"]; !ok {
			t.Errorf("Expected ref to %s from thing1",
				realPaths[0])
		}
		if _, ok := args.args["things"]; !ok {
			t.Errorf("Expected ref to %s from things",
				realPaths[0])
		}
		if _, ok := args.args["link"]; !ok {
			t.Errorf("Expected ref to %s from link",
				realPaths[0])
		}
	}
	if args := filesToArgs[realPaths[1]]; args == nil {
		t.Errorf("%s had no results.", realPaths[1])
	} else if len(args.args) != 2 {
		t.Errorf("Expected 2 refs for %s, got %d",
			realPaths[1], len(args.args))
	}
	if args := filesToArgs[realPaths[2]]; args == nil {
		t.Errorf("%s had no results.", realPaths[2])
	} else if len(args.args) != 1 {
		t.Errorf("Expected 1 ref for %s, got %d",
			realPaths[2], len(args.args))
	}
	if args := filesToArgs[realPaths[3]]; args == nil {
		t.Errorf("%s had no results.", realPaths[3])
	} else if len(args.args) != 1 {
		t.Errorf("Expected 1 ref for %s, got %d",
			realPaths[3], len(args.args))
	}
	if args := filesToArgs[realPaths[4]]; args == nil {
		t.Errorf("%s had no results.", realPaths[4])
	} else if args.args != nil {
		t.Errorf("Expected 0 refs for %s, got %d",
			realPaths[4], len(args.args))
	}
	things := path.Join(forkDir, "thing")
	if args := filesToArgs[things]; args == nil {
		t.Errorf("%s had no results.", things)
	} else if len(args.args) != 4 {
		t.Errorf("Expected 1 ref for %s, got %d",
			things, len(args.args))
	}
	if args := filesToArgs[link]; args == nil {
		t.Errorf("%s had no results.", link)
	} else {
		if len(args.args) != 3 {
			t.Errorf("Expected 1 ref for %s, got %d",
				link, len(args.args))
		}
		if _, ok := args.args["link"]; !ok {
			t.Errorf("Expected ref to %s from link",
				link)
		}
	}
	for _, p := range pathsList {
		if args := filesToArgs[p]; args == nil {
			t.Errorf("%s had no results.", p)
		} else if len(args.args) != 1 {
			t.Errorf("Expected 1 ref for %s, got %d",
				p, len(args.args))
		}
	}
}

func BenchmarkAddFilesToArgsMappings(b *testing.B) {
	forkDir, err := ioutil.TempDir("", "benchAddFilesToArgsMappings")
	if err != nil {
		b.Skip(err)
	}
	defer os.RemoveAll(forkDir)
	fileArgs, realPaths, outs, link, pathsList, err := makeStorageTestDir(b, forkDir)
	if err != nil {
		b.Fatal(err)
	}
	argToFiles := getArgsToFilesMap(fileArgs, outs, true, "test")
	if argToFiles == nil {
		b.Fatal("No argToFiles map")
	}
	filesToArgs := make(map[string]*vdrFileCache, len(fileArgs))
	addFilesToArgsMappings(forkDir, true, "test",
		filesToArgs, argToFiles)
	checkFileToArgMappings(b, filesToArgs, realPaths, forkDir, link, pathsList)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for key := range filesToArgs {
			delete(filesToArgs, key)
		}
		addFilesToArgsMappings(forkDir, true, "test",
			filesToArgs, argToFiles)
	}
}
