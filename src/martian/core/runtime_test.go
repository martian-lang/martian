//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian runtime tests.
//

package core

import (
	_ "encoding/json"
	_ "fmt"
	"os"
	_ "testing"
)

func MockRuntime() *Runtime {
	ENABLE_LOGGING = false // Disable core.LogInfo calls in Runtime
	return NewRuntime("local", "disable", "disable", os.Getenv("MROPATH"), "", "")
}

/*
func ExampleBuildCallSource() {
	rt := MockRuntime()

	// JSON bag as returned by argshim.
	jsonStr := `{"input_mode": "BCL_PROCESSOR", "trim_length": 10, "confident_regions": "/mnt/opt/meowmix/genometracks/hg19/human_conf_35.bed", "common_vars": "/mnt/opt/meowmix/variants/hg19/common/hg19.pickle", "targets_file": null, "sample_id": "3344", "sample_def": [{"gem_group": null, "sample_indices": ["TACTAGTC", "CTGCTCAT"], "lanes": null, "read_path": "/mnt/staging/stagesoc/pipestances/AB496/BCL_PROCESSOR_PD/AB496/1002.0.1/BCL_PROCESSOR_PD/BCL_PROCESSOR/DEMULTIPLEX/fork0/files/demultiplexed_fastq_path"}], "variant_results": null, "primers": ["P5:AATGATACGGCGACCACCGAGA", "P7RC:CAAGCAGAAGACGGCATACGAGAT", "Alt2-10N:AATGATACGGCGACCACCGAGATCTACACTAGATCGCTTGCTCATTCCCTACACGACGCTCTTCCGATCTNNNNNNNNNN", "R2RC:GTGACTGGAGTTCAGACGTGTGCTCTTCCGATCT", "R1-alt2:TTGCTCATTCCCTACACGACGCTCTTCCGATCT"], "lena_url": "lena-stagesoc", "exclude_non_bc_reads": false, "template_mass": 5, "genome": "hg19", "barcode_whitelist": null}`
	var v map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &v)

	src, err := rt.BuildCallSource([]string{"analyzer_pd.mro"}, "ANALYZER_PD", v)
	if err == nil {
		fmt.Println(src)
	} else {
		fmt.Println(err)
	}
	// Output:
	// @include "analyzer_pd.mro"
	//
	// call ANALYZER_PD(
	//     input_mode = "BCL_PROCESSOR",
	//     sample_def = [
	//         {
	//             "gem_group": null,
	//             "lanes": null,
	//             "read_path": "/mnt/staging/stagesoc/pipestances/AB496/BCL_PROCESSOR_PD/AB496/1002.0.1/BCL_PROCESSOR_PD/BCL_PROCESSOR/DEMULTIPLEX/fork0/files/demultiplexed_fastq_path",
	//             "sample_indices": [
	//                 "TACTAGTC",
	//                 "CTGCTCAT"
	//             ]
	//         }
	//     ],
	//     exclude_non_bc_reads = false,
	//     genome = "hg19",
	//     targets_file = null,
	//     confident_regions = "/mnt/opt/meowmix/genometracks/hg19/human_conf_35.bed",
	//     trim_length = 10,
	//     barcode_whitelist = null,
	//     primers = [
	//         "P5:AATGATACGGCGACCACCGAGA",
	//         "P7RC:CAAGCAGAAGACGGCATACGAGAT",
	//         "Alt2-10N:AATGATACGGCGACCACCGAGATCTACACTAGATCGCTTGCTCATTCCCTACACGACGCTCTTCCGATCTNNNNNNNNNN",
	//         "R2RC:GTGACTGGAGTTCAGACGTGTGCTCTTCCGATCT",
	//         "R1-alt2:TTGCTCATTCCCTACACGACGCTCTTCCGATCT"
	//     ],
	//     sample_id = "3344",
	//     lena_url = "lena-stagesoc",
	//     template_mass = 5.000000,
	//     common_vars = "/mnt/opt/meowmix/variants/hg19/common/hg19.pickle",
	// )
}
*/
