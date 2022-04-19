//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

/*
Generates a go source file declaring structs compatible with the tages
declared in the given mro sources.

MRO files are parsed given the current mropath.  If a specific set of stages
is not specified, then code is generated for all stages.

The given source file is created, with the given package name.  For each
stage which will be used, a structure is created which is appropriate for
serializing the stage args and outs files.  In addition, for stages which
split, structures are generated for stagedefs and chunk args.  These structs
are named as <stageName><File>, where file is one of Args, Outs, ChunkDef,
ChunkArgs, or JoinArgs.  Stages which do not split will have only the first
two of those.

ChunkDefs objects will have a ToChunkDef method, which converts from the stage-
specific chunk def object to a *core.ChunkDef, which is required by the go
adapter for the return value of the split.

ChunkArgs is a combination of ChunkDefs and the stage Args, and is used by the
chunk main to deserialize its arguments.

ChunkOuts is a combination of the split outs and the stage outs.  It defines
a custom json marshaller in order to ensure the outputs are correctly
flattened in the json representation.  It is used by the chunk main for its
output and by the join to deserialize the chunk outputs.

JoinArgs is a combination of the Args and to job resources structure.  It is
used by the join instead of Args if the join wants to see the thread/memory
request assigned to it by the split.

A stage without a split will look like the following simple example:

	func main(metadata *core.Metadata) (interface{}, error) {
		var args StageNameArgs
		if err := metadata.ReadInto(core.ArgsFile, &args); err != nil {
			return nil, err
		}
		return &StageNameOuts{
			Arg1: value1,
			Arg2: value2,
		}, nil
	}

Stages with splits will be more complex and should use the corresponding
datastructures.

Leading underscores are stripped from the stage.  The stage name is converted
to camelCase unless '-public' is specified on the command line, in which case
it is converted to PascalCase.

Given the input pipeline.mro:

	filetype bam;
	filetype vcf;
	filetype vcf.gz;
	filetype vcf.gz.tbi;
	filetype filter_params;
	filetype json;
	filetype bed;
	filetype tsv;
	filetype tsv.gz;
	filetype h5;
	filetype csv;

	stage POPULATE_INFO_FIELDS(
	    in  vcf      vc_precalled,
	    in  string   variant_mode,
	    in  vcf.gz   haploid_merge       "optional vcf to merge with normal calls",
	    in  string[] chunk_locus         "list of chunk loci, if supplying haploid_merge",
	    in  int      min_mapq_attach_bc,
	    out vcf.gz,
	    src py       "stages/snpindels/populate_info",
	) split using (
	    in  vcf      chunk_input,
	    out int      chunk_output,
	)

A user would run
	$ mro2go -package populate -o stagestructs.go pipeline.mro

to generate stagestructs.go:

	package populate

	import (
		"github.com/martian-lang/martian/martian/core"
	)

	// A structure to encode and decode args to the POPULATE_INFO_FIELDS stage.
	type PopulateInfoFieldsArgs struct {
		// vcf file
		VcPrecalled string `json:"vc_precalled"`
		VariantMode string `json:"variant_mode"`
		// vcf.gz file: optional vcf to merge with normal calls
		HaploidMerge string `json:"haploid_merge"`
		// list of chunk loci, if supplying haploid_merge
		ChunkLocus      []string `json:"chunk_locus"`
		MinMapqAttachBc int      `json:"min_mapq_attach_bc"`
	}

	// A structure to encode and decode outs from the POPULATE_INFO_FIELDS stage.
	type PopulateInfoFieldsOuts struct {
		// vcf.gz file
		Default string `json:"default"`
	}

	// A structure to encode and decode args to the POPULATE_INFO_FIELDS chunks.
	// Defines the resources and arguments of a chunk.
	type PopulateInfoFieldsChunkDef struct {
		*core.JobResources `json:",omitempty"`
		// vcf file
		ChunkInput string `json:"chunk_input"`
	}

	func (self *PopulateInfoFieldsChunkIns) ToChunkDef() *core.ChunkDef {
		return &core.ChunkDef{
			Args: core.ArgumentMap{
				"chunk_input": self.ChunkInput,
			},
			Resources: self.JobResources,
		}
	}

	// A structure to decode args to the chunks
	type PopulateInfoFieldsChunkArgs struct {
		PopulateInfoFieldsChunkDef
		PopulateInfoFieldsArgs
	}

	// A structure to decode args to the join method.
	type PopulateInfoFieldsJoinArgs struct {
		core.JobResources
		PopulateInfoFieldsArgs
	}

	// A structure to encode outs from the chunks.
	type PopulateInfoFieldsChunkOuts struct {
		PopulateInfoFieldsOuts
		ChunkOutput int `json:"chunk_output"`
	}

	func (self *PopulateInfoFieldsChunkOuts) MarshalJSON() ([]byte, error) {
		...
	}
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

func main() {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: %s [options] <source.mro> [source2.mro...]\n", os.Args[0])
		flags.PrintDefaults()
	}
	outfile := flags.String("output", "",
		"The destination file name.  The default is <basename of source>.go")
	flags.StringVar(outfile, "o", "",
		"The destination file name.  The default is <basename of source>.go")
	outDir := flags.String("output-dir", "",
		"The destination directory, for generating multiple .mro files. "+
			"Files will be named by the source mro basename, with the "+
			"extension changed to .go.")
	packageName := flags.String("package", "",
		"The name of the package in for the generated source file.  "+
			"Defaults to the name of the output file's directory.")
	flags.StringVar(packageName, "p", "",
		"The name of the package in for the generated source file.  "+
			"Defaults to the name of the output file's directory.")
	stageNames := flags.String("stage", "",
		"Only generate code for the given stages (comma-separated list).")
	pipelineNames := flags.String("pipeline", "",
		"Only generate structs for the given pipelines (comma-separated list).")
	structs := flags.Bool("structs", true,
		"Also generate any structs required for input/output parameters.")
	stdout := flags.Bool("stdout", false,
		"Write the go source to standard out.")
	onlyIns := flags.Bool("input-only", false,
		"If set, only create structs for inputs.")
	if err := flags.Parse(os.Args[1:]); err != nil {
		// ExitOnError should mean that it never returns an error.
		panic(err)
	}
	if flags.NArg() < 1 {
		flags.Usage()
		os.Exit(1)
	}
	if *pipelineNames != "" {
		if *stageNames != "" {
			fmt.Fprintf(os.Stderr,
				"-stage and -pipeline are incompatible.")
			os.Exit(1)
		}
		*stageNames = *pipelineNames
	}
	// Require strict enforcement of mro language.  This prevents, for
	// example, chunk in parameters with names which duplicate stage ins,
	// which would break these datastructures.
	syntax.SetEnforcementLevel(syntax.EnforceError)
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Could not get working directory: %v\n",
			err)
	}
	mroPaths := append([]string{cwd},
		util.ParseMroPath(os.Getenv("MROPATH"))...)
	var f *os.File
	if *stdout {
		f = os.Stdout
	} else if *outfile != "" {
		if *outDir != "" {
			fmt.Fprintln(os.Stderr,
				"Specifying an output file name is incompatible with "+
					"specifying an output directory.")
		}
		if flags.NArg() > 1 {
			fmt.Fprintln(os.Stderr,
				"Writing multiple mro files to a single output go file is not supported.")
			os.Exit(1)
		}
		if t, err := os.Create(*outfile); err != nil {
			fmt.Fprintf(os.Stderr,
				"Error opening destination file %s: %v\n",
				*outfile, err)
			os.Exit(1)
		} else {
			f = t
			defer func() {
				if err := f.Close(); err != nil {
					fmt.Fprintf(os.Stderr,
						"Error closing %s: %v\n",
						*outfile, err)
					os.Exit(1)
				}
			}()
		}
	}
	stageNamesList := maybeSplitList(*stageNames)
	var seenStructs map[string]struct{}
	var lastPackage string
	for _, mrofile := range flags.Args() {
		thisPackage := *packageName
		if thisPackage == "" {
			thisOut := *outfile
			if thisOut == "" {
				thisOut = filepath.Join(*outDir, ".go")
			}
			if p, err := filepath.Abs(thisOut); err != nil {
				fmt.Fprintf(os.Stderr,
					"Error detecting directory name: %v\n", err)
				os.Exit(1)
			} else {
				thisPackage = path.Base(path.Dir(p))
			}
		}
		if *structs && (lastPackage != thisPackage || seenStructs == nil) {
			seenStructs = make(map[string]struct{})
			lastPackage = thisPackage
		}
		if *outDir != "" {
			bn := filepath.Base(mrofile)
			bn = strings.TrimSuffix(bn, filepath.Ext(bn)) + ".go"
			f, err = os.Create(filepath.Join(*outDir, bn))
			if err != nil {
				fmt.Fprintf(os.Stderr,
					"Error opening destination file %s: %v\n",
					*outfile, err)
				os.Exit(1)
			}
		}
		processFile(f, mrofile, thisPackage, stageNamesList,
			mroPaths, *pipelineNames != "", *onlyIns, seenStructs)
		if *outDir != "" {
			if err := f.Close(); err != nil {
				fmt.Fprintf(os.Stderr,
					"Error closing %s: %v\n",
					*outfile, err)
				os.Exit(1)
			}
		}
	}
}

func maybeSplitList(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func processFile(dest *os.File, mrofile, packageName string, stageNames []string,
	mroPaths []string, pipeline, onlyIns bool,
	seenStructs map[string]struct{}) {
	if dest == nil {
		thisOut := path.Base(strings.TrimSuffix(mrofile, ".mro")) + ".go"
		if t, err := os.Create(thisOut); err != nil {
			fmt.Fprintf(os.Stderr,
				"Error opening destination file %s: %v\n",
				thisOut, err)
			os.Exit(1)
		} else {
			dest = t
			defer func() {
				if err := dest.Close(); err != nil {
					fmt.Fprintf(os.Stderr,
						"Error closing %s: %v\n",
						thisOut, err)
					os.Exit(1)
				}
			}()
		}
	}
	if src, _, err := readSrc(mrofile, mroPaths); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading source file\n%s\n", err.Error())
		os.Exit(1)
	} else if err := MroToGo(dest, src,
		mrofile, stageNames, mroPaths,
		packageName, dest.Name(), pipeline, onlyIns, seenStructs); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating go source for %s\n%s\n",
			mrofile, err.Error())
		os.Exit(1)
	}
}

func readSrc(mrofile string, mroPaths []string) ([]byte, string, error) {
	if mrofile == "-" {
		src, err := ioutil.ReadAll(os.Stdin)
		return src, "<stdin>", err
	} else if _, err := os.Stat(mrofile); err == nil {
		src, err := ioutil.ReadFile(mrofile)
		return src, mrofile, err
	} else if os.IsNotExist(err) {
		if p, found := util.SearchPaths(mrofile, mroPaths); !found {
			return nil, mrofile, err
		} else {
			src, err := ioutil.ReadFile(p)
			return src, p, err
		}
	} else {
		return nil, mrofile, err
	}
}

func MroToGo(dest io.Writer,
	src []byte, mrofile string, stageNames, mroPaths []string,
	pkg, outName string, pipeline, onlyIns bool,
	seenStructs map[string]struct{}) error {
	if ast, err := parseMro(src, mrofile, mroPaths); err != nil {
		return err
	} else {
		return gofmt(dest,
			makeCallableGoRaw(ast, pkg, mrofile, stageNames,
				pipeline, onlyIns, seenStructs), outName)
	}
}

func parseMro(src []byte, fname string, mroPaths []string) (*syntax.Ast, error) {
	_, _, ast, err := syntax.ParseSourceBytes(src, fname, mroPaths, false)
	return ast, err
}

func getCallables(ast *syntax.Ast, fname string,
	pipelineNames []string, pipeline bool) []syntax.Callable {
	if pipeline {
		return getPipelines(ast, fname, pipelineNames)
	} else {
		return getStages(ast, fname, pipelineNames)
	}
}

func matchAny(id string, names []string) bool {
	for _, n := range names {
		if n == id {
			return true
		}
	}
	return len(names) == 0
}

func getPipelines(ast *syntax.Ast, fname string, pipelineNames []string) []syntax.Callable {
	stages := make([]syntax.Callable, 0, len(ast.Pipelines))
	for _, p := range ast.Pipelines {
		if path.Base(p.Node.Loc.File.FullPath) == path.Base(fname) &&
			matchAny(p.GetId(), pipelineNames) {
			stages = append(stages, p)
		}
	}
	return stages
}

func getStages(ast *syntax.Ast, fname string, stageNames []string) []syntax.Callable {
	stages := make([]syntax.Callable, 0, len(ast.Stages))
	for _, stage := range ast.Stages {
		if path.Base(stage.Node.Loc.File.FullPath) == path.Base(fname) &&
			matchAny(stage.GetId(), stageNames) {
			stages = append(stages, stage)
		}
	}
	return stages
}
