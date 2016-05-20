// Copyright (c) 2016 10X Genomics, Inc. All rights reserved.

package main

import (
	"flag"
	"martian/core"
	"os"
	"strings"
)

var input_new_mro = flag.String("mro", "", "MRO invocation file for new pipestance")
var input_new_psid = flag.String("psid", "", "Pipestance ID for new pipestance")
var old_pipestance_path = flag.String("base", "", "Base (original) pipestance path")
var invalidated_stages = flag.String("inv", "", "Comma separated list of changed stages")
var jobmode = flag.String("jobmode", "local", "job mode (sge, local, etc)");

func main() {
	flag.Parse()

	if *input_new_mro == "" || *input_new_psid == "" || *old_pipestance_path == "" || *invalidated_stages == "" {
		flag.PrintDefaults()
		os.Exit(1)

	}

	/* Setup variables for the old pipestance */
	var oldi core.PipestanceSetup

	oldi.Srcpath = *old_pipestance_path + "/_mrosource"
	oldi.Psid = "x"
	oldi.PipestancePath = *old_pipestance_path
	//oldi.MroPaths = core.ParseMroPath(*old_pipestance_path)
	oldi.MroVersion = "x"
	oldi.Envs = map[string]string{}
	oldi.JobMode = "local"

	/* Setup variables for the new pipestance */
	var newi core.PipestanceSetup

	newi.Srcpath = *input_new_mro
	newi.PipestancePath = *input_new_psid
	newi.Psid = *input_new_psid
	//newi.MroPaths= strings.Split(*new_mro_paths, ":");
	newi.MroPaths = core.ParseMroPath(os.Getenv("MROPATH"))
	newi.MroVersion = "y"
	newi.Envs = map[string]string{}
	newi.JobMode = *jobmode;

	invalidated_stages_a := strings.Split(*invalidated_stages, ",")

	core.MRTBuildPipeline(&newi, &oldi, invalidated_stages_a)

	core.Println("DONE! To run your pipeline say: mrp %v %v", *input_new_mro, *input_new_psid);

	
}
