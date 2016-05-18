
package main

import (
	"martian/core"
	"flag"
	"strings"
)

var input_new_mro = flag.String("mro", "", "Invocation for new pipestance")
var input_new_psid = flag.String("psid", "", "Pipestance ID for new pipestance")
var new_mro_paths = flag.String("mropath", "./", "Search string for MROs");

var old_pipestance_path = flag.String("base", "", "Base pipestance path")

var invalidated_stages = flag.String("inv", "", "Stages i changed");




func main() {
	flag.Parse();

	var oldi core.PSInfo;

	oldi.Srcpath = *old_pipestance_path + "/_mrosource";
	oldi.Psid = "x";
	oldi.PipestancePath = *old_pipestance_path
	//oldi.MroPaths = []string{}
	// This makes it work but is like totally wrong n stuff *** XXX YYY ZZZ XXX YYY ZZZ
	oldi.MroPaths = core.ParseMroPath(*new_mro_paths);
	oldi.MroVersion="x"
	oldi.Envs = map[string]string{}

	var newi core.PSInfo;

	newi.Srcpath = *input_new_mro;
	newi.PipestancePath = *input_new_psid
	newi.Psid = *input_new_psid;
	//newi.MroPaths= strings.Split(*new_mro_paths, ":");
	newi.MroPaths = core.ParseMroPath(*new_mro_paths);
	newi.MroVersion="y"
	newi.Envs = map[string]string{}

	invalidated_stages_a := strings.Split(*invalidated_stages, ",");
	core.Println("O: %v", oldi);
	core.Println("N: %v", newi);

	core.DoIt(&newi, &oldi, invalidated_stages_a)
	



}

