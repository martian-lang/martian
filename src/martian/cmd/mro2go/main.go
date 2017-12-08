//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
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
		"martian/core"
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
	"martian/syntax"
	"martian/util"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	packageName := flags.String("package", "",
		"The name of the package in for the generated source file.  "+
			"Defaults to the name of the output file's directory.")
	flags.StringVar(packageName, "p", "",
		"The name of the package in for the generated source file.  "+
			"Defaults to the name of the output file's directory.")
	stageName := flags.String("stage", "",
		"Only generate code for the given stage.")
	stdout := flags.Bool("stdout", false,
		"Write the go source to standard out.")
	if err := flags.Parse(os.Args[1:]); err != nil {
		// ExitOnError should mean that it never returns an error.
		panic(err)
	}
	if flags.NArg() < 1 {
		flags.Usage()
		os.Exit(1)
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
		if flags.NArg() > 1 {
			fmt.Fprintf(os.Stderr,
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
	for _, mrofile := range flags.Args() {
		thisPackage := *packageName
		if thisPackage == "" {
			thisOut := *outfile
			if thisOut == "" {
				thisOut = ".go"
			}
			if p, err := filepath.Abs(thisOut); err != nil {
				fmt.Fprintf(os.Stderr,
					"Error detecting directory name: %v\n", err)
				os.Exit(1)
			} else {
				thisPackage = path.Base(path.Dir(p))
			}
		}
		processFile(f, mrofile, *stageName, thisPackage, mroPaths)
	}
}

func processFile(dest *os.File, mrofile, stageName, packageName string,
	mroPaths []string) {
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
	} else if err := MroToGo(dest, string(src),
		mrofile, stageName, mroPaths,
		packageName, dest.Name()); err != nil {
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
	src, mrofile, stageName string, mroPaths []string,
	pkg, outName string) error {
	if ast, err := parseMro(src, mrofile, mroPaths); err != nil {
		return err
	} else {
		return gofmt(dest, makeGoRaw(ast, pkg, mrofile, stageName), outName)
	}
}

func parseMro(src, fname string, mroPaths []string) (*syntax.Ast, error) {
	_, _, ast, err := syntax.ParseSource(src, fname, mroPaths, false)
	return ast, err
}

func getStages(ast *syntax.Ast, fname, stageName string) []*syntax.Stage {
	stages := make([]*syntax.Stage, 0, len(ast.Stages))
	for _, stage := range ast.Stages {
		if path.Base(stage.Node.Fname) == path.Base(fname) &&
			(stageName == "" || stage.Id == stageName) {
			stages = append(stages, stage)
		}
	}
	return stages
}
