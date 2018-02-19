// Copyright (c) 2016 10X Genomics, Inc. All rights reserved.

/*
 * mrt_helper is mart of the MaRtian Test system to re-running part of a
 * pipeline.
 *
 * Given a old pipestance, and a list of changes stages, it create a new
 * pipestance directory with unaffected stages linked to the original
 * pipestance directory.  The new pipestance may then be restarted with
 * mrp and only the changes stages (and their dependencies) will be
 * rerun.
 */

package main

import (
	"flag"
	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/util"
	"os"
	"strings"
)

var input_new_mro = flag.String("mro", "", "MRO invocation file for new pipestance")
var input_new_psid = flag.String("psid", "", "Pipestance ID for new pipestance")
var old_pipestance_path = flag.String("base", "", "Base (original) pipestance path")
var invalidated_stages = flag.String("inv", "", "Comma separated list of changed stages")
var jobmode = flag.String("jobmode", "local", "job mode (sge, local, etc)")

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
	oldi.MroVersion = "x"
	oldi.Envs = map[string]string{}
	oldi.JobMode = "local"

	/* Setup variables for the new pipestance */
	var newi core.PipestanceSetup

	newi.Srcpath = *input_new_mro
	newi.PipestancePath = *input_new_psid
	newi.Psid = *input_new_psid
	newi.MroPaths = util.ParseMroPath(os.Getenv("MROPATH"))
	newi.MroVersion, _ = util.GetMroVersion(newi.MroPaths)
	newi.Envs = map[string]string{}
	newi.JobMode = *jobmode

	/* Parse the list of stages to be invalidated */
	invalidated_stages_a := strings.Split(*invalidated_stages, ",")

	/* Build the new pipeline */
	core.MRTBuildPipeline(&newi, &oldi, invalidated_stages_a)

	util.Println("DONE! To run your pipeline say: mrp %v %v", *input_new_mro, *input_new_psid)

}
