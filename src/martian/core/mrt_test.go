package core

import (
	"testing"
	"os"
)

func TestMRT1(t * testing.T) {

	cwd, _:= os.Getwd();

	mroPaths := ParseMroPath(cwd + "/test_data");
	mroVersion, _ := GetMroVersion(mroPaths);
	psid :="hello_world";
	envs := make(map[string]string);
	srcpath := "/Users/dstaff/code/martian/src/martian/core/test_data/call1.mro";
	pipestancepath := cwd + "/test_data/ds13"


	a1 := PSInfo{
		srcpath,
		psid,
		pipestancepath,
		mroPaths,
		mroVersion,
		envs}

	a2:=a1;
	a2.pipestancePath="squeeeek";

	DoIt(&a2, &a1);
}
