// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package core // import "github.com/martian-lang/martian/martian/core"

// Martian runtime. This contains the code to instantiate or restart
// pipestances.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime/trace"
	"strings"
	"time"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

const (
	STAGE_TYPE_SPLIT = "split"
	STAGE_TYPE_CHUNK = "chunk"
	STAGE_TYPE_JOIN  = "join"
)

const forkPrintInterval = 5 * time.Minute

// Helpers

func parseTimestamp(data string) string {
	// Backwards compatible with current and plain timestamp formats
	timestamp := strings.Split(data, "\n")[0]
	prefix := "start:"
	if strings.HasPrefix(timestamp, prefix) {
		timestamp = timestamp[len(prefix):]
		return strings.TrimSpace(timestamp)
	}
	return timestamp
}

func ParseVersions(data string) (string, string, error) {
	var versions VersionInfo
	if err := json.Unmarshal([]byte(data), &versions); err != nil {
		return "", "", err
	}
	return versions.Martian, versions.Pipelines, nil
}

func VerifyVDRMode(vdrMode VdrMode) {
	switch vdrMode {
	case VdrRolling, VdrPost, VdrDisable, VdrStrict:
		return
	}
	util.PrintInfo("runtime",
		"Invalid VDR mode: %s. Valid VDR modes: rolling, post, disable, strict",
		vdrMode)
	os.Exit(1)
}

func VerifyOnFinish(onfinish string) {
	if _, err := exec.LookPath(onfinish); err != nil {
		util.PrintInfo("runtime",
			"Invalid onfinish hook executable (%v): %v",
			err, onfinish)
		os.Exit(1)
	}
}

// Reads config file for regexps which, when matched, indicate that
// an error is likely transient.
func getRetryRegexps() (retryOn []*regexp.Regexp, defaultRetries int) {
	retryfile := util.RelPath(path.Join("..", "jobmanagers", "retry.json"))

	if _, err := os.Stat(retryfile); os.IsNotExist(err) {
		return []*regexp.Regexp{
			regexp.MustCompile("^signal: "),
		}, 0
	}
	type retryJson struct {
		DefaultRetries int      `json:"default_retries"`
		RetryOn        []string `json:"retry_on"`
	}
	bytes, err := ioutil.ReadFile(retryfile)
	if err != nil {
		util.PrintInfo("runtime", "Retry config file could not be loaded:\n%v\n", err)
		os.Exit(1)
	}
	var retryInfo *retryJson
	if err = json.Unmarshal(bytes, &retryInfo); err != nil {
		util.PrintInfo("runtime", "Retry config file could not be parsed:\n%v\n", err)
		os.Exit(1)
	}
	regexps := make([]*regexp.Regexp, len(retryInfo.RetryOn))
	for i, exp := range retryInfo.RetryOn {
		regexps[i] = regexp.MustCompile(exp)
	}
	return regexps, retryInfo.DefaultRetries
}

func DefaultRetries() int {
	_, def := getRetryRegexps()
	return def
}

//=============================================================================
// Runtime
//=============================================================================

const disable = "disable"

type VdrMode string

const (
	VdrDisable = disable
	VdrPost    = "post"
	VdrRolling = "rolling"
	VdrStrict  = "strict"
)

// Configuration required to initialize a Runtime object.
type RuntimeOptions struct {
	// The runtime mode (required): either "local" or a named mode from
	// jobmanagers/config.json
	JobMode string

	// The volatile disk recovery mode (required): either "post",
	// "rolling", "strict", or "disable".
	VdrMode VdrMode

	// The profiling mode (required): "disable" or one of the available
	// constants.
	ProfileMode     ProfileMode
	MartianVersion  string
	LocalMem        int
	LocalVMem       int
	LocalCores      int
	MemPerCore      int
	MaxJobs         int
	JobFreqMillis   int
	ResourceSpecial string
	FullStageReset  bool
	StackVars       bool
	Zip             bool
	SkipPreflight   bool
	Monitor         bool
	Debug           bool
	StressTest      bool
	OnFinishHandler string
	Overrides       *PipestanceOverrides
	LimitLoadavg    bool
	NeverLocal      bool
}

const localMode = "local"

func DefaultRuntimeOptions() RuntimeOptions {
	return RuntimeOptions{
		MartianVersion: util.GetVersion(),
		ProfileMode:    DisableProfile,
		JobMode:        localMode,
		VdrMode:        VdrRolling,
	}
}

// returns the set of command line flags which would set these options.
func (config *RuntimeOptions) ToFlags() []string {
	var flags []string
	if config.JobMode != localMode {
		flags = append(flags, "--jobmode="+config.JobMode)
	}
	if config.VdrMode != VdrRolling {
		flags = append(flags, "--vdrmode="+string(config.VdrMode))
	}
	if config.ProfileMode != DisableProfile {
		flags = append(flags, fmt.Sprintf("--profile=%v",
			config.ProfileMode))
	}
	if config.LocalMem != 0 {
		flags = append(flags, fmt.Sprintf("--localmem=%d",
			config.LocalMem))
	}
	if config.LocalVMem != 0 {
		flags = append(flags, fmt.Sprintf("--localvmem=%d",
			config.LocalVMem))
	}
	if config.LocalCores != 0 {
		flags = append(flags, fmt.Sprintf("--localcores=%d",
			config.LocalCores))
	}
	if config.MemPerCore != 0 {
		flags = append(flags, fmt.Sprintf("--mempercore=%d",
			config.MemPerCore))
	}
	if config.MaxJobs != 0 {
		flags = append(flags, fmt.Sprintf("--maxjobs=%d",
			config.MaxJobs))
	}
	if config.JobFreqMillis != 0 {
		flags = append(flags, fmt.Sprintf("--jobinterval=%d",
			config.JobFreqMillis))
	}
	if config.StackVars {
		flags = append(flags, "--stackvars")
	}
	if config.Zip {
		flags = append(flags, "--zip")
	}
	if config.SkipPreflight {
		flags = append(flags, "--nopreflight")
	}
	if config.Monitor {
		flags = append(flags, "--monitor")
	}
	if config.Debug {
		flags = append(flags, "--debug")
	}
	if config.StressTest {
		flags = append(flags, "--stest")
	}
	if config.OnFinishHandler != "" {
		if p, err := exec.LookPath(config.OnFinishHandler); err != nil {
			util.LogError(err, "runtime",
				"Could not find path for onfinish handler.")
			flags = append(flags, "--onfinish="+config.OnFinishHandler)
		} else if ap, err := filepath.Abs(p); err != nil {
			util.LogError(err, "runtime",
				"Could not find abs path for onfinish handler.")
			flags = append(flags, "--onfinish="+p)
		} else {
			flags = append(flags, "--onfinish="+ap)
		}
	}
	if config.LimitLoadavg {
		flags = append(flags, "--limit-loadavg")
	}
	if config.NeverLocal {
		flags = append(flags, "--never-local")
	}
	return flags
}

// Collects configuration and state required to initialize and run pipestances
// and stagestances.
type Runtime struct {
	Config          *RuntimeOptions
	adaptersPath    string
	mrjob           string
	JobManager      JobManager
	LocalJobManager *LocalJobManager
	overrides       *PipestanceOverrides
	jobConfig       *JobManagerJson
}

func (c *RuntimeOptions) NewRuntime() *Runtime {
	self := &Runtime{
		Config:       c,
		adaptersPath: util.RelPath(path.Join("..", "adapters")),
		mrjob:        util.RelPath("mrjob"),
	}

	self.jobConfig = getJobConfig(c.ProfileMode)
	self.LocalJobManager = NewLocalJobManager(c.LocalCores,
		c.LocalMem, c.LocalVMem,
		c.Debug,
		c.LimitLoadavg,
		c.JobMode != localMode,
		self.jobConfig)
	if c.JobMode == localMode {
		self.JobManager = self.LocalJobManager
	} else {
		self.JobManager = NewRemoteJobManager(c.JobMode, c.MemPerCore, c.MaxJobs,
			c.JobFreqMillis, c.ResourceSpecial, self.jobConfig, c.Debug)
	}
	VerifyVDRMode(c.VdrMode)

	if c.Overrides == nil {
		self.overrides, _ = ReadOverrides("")
	} else {
		self.overrides = c.Overrides
	}

	return self
}

// Instantiate a pipestance object given a psid, MRO source, and a
// pipestance path. This is the core (private) method called by the
// public InvokeWithSource and Reattach methods.
func (self *Runtime) instantiatePipeline(src []byte, srcPath, psid,
	pipestancePath string, mroPaths []string, mroVersion string,
	envs map[string]string, checkSrc, readOnly bool,
	ctx context.Context) (string, *syntax.Ast, *Pipestance, error) {
	r := trace.StartRegion(ctx, "instantiatePipeline")
	defer r.End()
	// Parse the invocation source.
	postsrc, _, ast, err := syntax.ParseSourceBytes(src, srcPath, mroPaths, checkSrc)
	if err != nil {
		return "", nil, nil, err
	}

	// Check there's a call.
	if ast.Call == nil {
		return "", nil, nil, &RuntimeError{
			Msg: "cannot start a pipeline without a call statement",
		}
	}
	// Make sure it's a pipeline we're calling.
	if pipeline := ast.Callables.Table[ast.Call.DecId]; pipeline == nil {
		return "", nil, nil, &RuntimeError{
			Msg: fmt.Sprintf("'%s' is not a declared pipeline", ast.Call.DecId),
		}
	}

	invocationData, _ := BuildDataForAst(ast)

	// Instantiate the pipeline.
	if !readOnly {
		if err := CheckMinimalSpace(pipestancePath); err != nil {
			return "", nil, nil, err
		}
	}
	callGraph, err := ast.MakePipelineCallGraph("ID."+psid+".", ast.Call)
	if err != nil {
		return "", nil, nil, err
	}
	var srcPaths []string
	if checkSrc || !readOnly {
		srcPaths = append(mroPaths,
			filepath.SplitList(os.Getenv("PATH"))...)
	}
	pipestance, err := NewPipestance(
		NewTopNode(self, callGraph.GetFqid()[:3+len(psid)], pipestancePath,
			mroPaths, mroVersion,
			envs, invocationData,
			&ast.TypeTable),
		callGraph, srcPaths)
	if err != nil {
		return "", nil, nil, err
	}

	// Lock the pipestance if not in read-only mode.
	if !readOnly {
		if err := pipestance.Lock(); err != nil {
			return "", nil, nil, err
		}
		err = pipestance.getNode().mkdirs()
	}

	ast.TypeTable.Freeze()
	return postsrc, ast, pipestance, err
}

// Invokes a new pipestance.
func (self *Runtime) InvokePipeline(src string, srcPath string, psid string,
	pipestancePath string, mroPaths []string, mroVersion string,
	envs map[string]string, tags []string) (*Pipestance, error) {
	// Error if pipestance directory is non-empty, otherwise create.
	if err := os.MkdirAll(pipestancePath, 0777); err != nil {
		return nil, err
	}
	if fileNames, err := util.Readdirnames(pipestancePath); err != nil {
		return nil, err
	} else {
		for _, name := range fileNames {
			if len(name) > 0 && name[0] != '.' {
				return nil, &PipestanceExistsError{psid}
			}
		}
	}

	// Expand env vars in invocation source and instantiate.
	src = os.ExpandEnv(src)
	readOnly := false
	postsrc, _, pipestance, err := self.instantiatePipeline([]byte(src), srcPath, psid,
		pipestancePath, mroPaths,
		mroVersion, envs, false, readOnly, context.Background())
	if err != nil {
		// If instantiation failed, delete the pipestance folder.
		os.RemoveAll(pipestancePath)
		return nil, err
	}

	// Write top-level metadata files.
	if err := pipestance.metadata.WriteRaw(InvocationFile, src); err != nil {
		os.RemoveAll(pipestancePath)
		return pipestance, err
	}
	if err := pipestance.metadata.WriteRaw(JobModeFile,
		self.Config.JobMode); err != nil {
		os.RemoveAll(pipestancePath)
		return pipestance, err
	}
	if err := pipestance.metadata.WriteRaw(MroSourceFile, postsrc); err != nil {
		os.RemoveAll(pipestancePath)
		return pipestance, err
	}
	if err := pipestance.metadata.Write(VersionsFile, &VersionInfo{
		Martian:   self.Config.MartianVersion,
		Pipelines: mroVersion,
	}); err != nil {
		os.RemoveAll(pipestancePath)
		return pipestance, err
	}
	if err := pipestance.metadata.Write(TagsFile, tags); err != nil {
		os.RemoveAll(pipestancePath)
		return pipestance, err
	}
	if uid := os.Getenv("MRO_FORCE_UUID"); uid == "" {
		if err := pipestance.SetUuid(NewUUID().String()); err != nil {
			os.RemoveAll(pipestancePath)
			return pipestance, err
		}
	} else {
		util.LogInfo("runtime", "UUID forced to %s by environment", uid)
		if err := pipestance.SetUuid(uid); err != nil {
			os.RemoveAll(pipestancePath)
			return pipestance, err
		}
	}
	if err := pipestance.metadata.WriteRaw(TimestampFile,
		"start: "+util.Timestamp()); err != nil {
		os.RemoveAll(pipestancePath)
		return pipestance, err
	}

	return pipestance, nil
}

func (self *Runtime) ReattachToPipestance(psid string, pipestancePath string,
	src string, invocationPath string, mroPaths []string,
	mroVersion string, envs map[string]string, checkSrc bool, readOnly bool,
	ctx context.Context) (*Pipestance, error) {
	return self.reattachToPipestance(psid, pipestancePath,
		src, invocationPath, mroPaths,
		mroVersion, envs, checkSrc,
		readOnly, InvocationFile,
		ctx)
}

func (self *Runtime) ReattachToPipestanceWithMroSrc(psid string, pipestancePath string,
	src string, invocationPath string, mroPaths []string,
	mroVersion string, envs map[string]string, checkSrc bool,
	readOnly bool, ctx context.Context) (*Pipestance, error) {
	return self.reattachToPipestance(psid, pipestancePath,
		src, invocationPath, mroPaths,
		mroVersion, envs, checkSrc,
		readOnly, MroSourceFile, ctx)
}

// Reattaches to an existing pipestance.
func (self *Runtime) reattachToPipestance(psid string, pipestancePath string,
	srcStr string, invocationPath string, mroPaths []string,
	mroVersion string, envs map[string]string, checkSrc bool, readOnly bool,
	srcType MetadataFileName, ctx context.Context) (*Pipestance, error) {
	var src []byte
	if srcStr == "" {
		if invocationPath == "" {
			invocationPath = path.Join(pipestancePath, srcType.FileName())
		}
		if data, err := ioutil.ReadFile(invocationPath); err != nil {
			return nil, &PipestancePathError{pipestancePath}
		} else {
			src = data
		}
	} else {
		src = []byte(srcStr)
	}
	if checkSrc {
		// Read in the existing _invocation file.
		data, err := ioutil.ReadFile(path.Join(pipestancePath, srcType.FileName()))
		if err != nil {
			return nil, &PipestancePathError{pipestancePath}
		}
		// Check if _invocation has changed.
		if !bytes.Equal(src, data) {
			return nil, &PipestanceInvocationError{psid, invocationPath}
		}
	}
	// Instantiate the pipestance.
	_, ast, pipestance, err := self.instantiatePipeline(
		src, invocationPath,
		psid, pipestancePath, mroPaths,
		mroVersion, envs, checkSrc, readOnly, ctx)
	if err != nil {
		return nil, err
	}
	if checkSrc && srcType != MroSourceFile {
		oldSrcFile := path.Join(pipestancePath, MroSourceFile.FileName())
		if _, _, oldAst, err := syntax.Compile(oldSrcFile, mroPaths, false); err != nil {
			if !readOnly {
				pipestance.Unlock()
			}
			return nil, err
		} else if !ast.EquivalentCall(oldAst) {
			if !readOnly {
				pipestance.Unlock()
			}
			return nil, &PipestanceInvocationError{psid, invocationPath}
		}
	}

	// If _jobmode exists, make sure we reattach to pipestance in the same job mode.
	if !readOnly {
		if err := pipestance.VerifyJobMode(); err != nil {
			pipestance.Unlock()
			return nil, err
		}
	}

	// If _metadata exists, unzip it so the pipestance can read its metadata.
	metadataPath := path.Join(pipestancePath, MetadataZip.FileName())
	if _, err := os.Stat(metadataPath); err == nil {
		if err := util.UnzipIgnoreExisting(metadataPath); err != nil {
			if !readOnly {
				pipestance.Unlock()
			}
			return nil, err
		}
		os.Remove(metadataPath)
	}
	pipestance.RestoreForks(ctx)

	// If we're reattaching in local mode, restart any stages that were
	// left in a running state from last mrp run. The actual job would
	// have been killed by the CTRL-C or, if not, by SIGTERM when the
	// mrp process died (on OSes where pdeathsig is supported).
	if !readOnly {
		util.PrintInfo("runtime", "Reattaching in %s mode.", self.Config.JobMode)
		if err := pipestance.RestartRunningNodes(self.Config.JobMode, ctx); err != nil {
			pipestance.Unlock()
			return nil, err
		}
	}

	return pipestance, nil
}

func (self *Runtime) GetSerializationInto(pipestancePath string,
	name MetadataFileName, target interface{}) error {
	metadata := NewMetadata("", pipestancePath)
	return metadata.ReadInto(name, target)
}

func (self *Runtime) GetSerialization(pipestancePath string, name MetadataFileName) (LazyArgumentMap, bool) {
	metadata := NewMetadata("", pipestancePath)
	metadata.loadCache()
	if metadata.exists(name) {
		if d, err := metadata.read(name, self.FreeMemBytes()/2); err != nil {
			return nil, false
		} else {
			return d, true
		}
	}
	return nil, false
}

func (self *Runtime) GetMetadata(pipestancePath string, metadataPath string) (io.ReadCloser, error) {
	metadata := NewMetadata("", pipestancePath)
	metadata.loadCache()
	if mdf := MetadataFileName(
		strings.TrimPrefix(metadataPath, MetadataFilePrefix)); metadata.exists(mdf) {
		return metadata.openFile(mdf)
	}
	if !filepath.IsAbs(metadataPath) {
		metadataPath = path.Join(pipestancePath, metadataPath)
	}
	if metadata.exists(MetadataZip) {
		relPath, _ := filepath.Rel(pipestancePath, metadataPath)

		// Relative paths outside the pipestance directory will be ignored.
		if !strings.Contains(relPath, "..") {
			if data, err := util.ReadZipFile(metadata.MetadataFilePath(MetadataZip), relPath); err == nil {
				return data, nil
			}
		}
	}
	data, err := os.Open(metadataPath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (self *Runtime) freeMemMB() int64 {
	if !self.Config.Monitor {
		return 0
	}
	if free := self.LocalJobManager.memMBSem.CurrentSize(); free < 1024 {
		return free
	} else {
		return 1024
	}
}

func (self *Runtime) ProfileConfig(mode ProfileMode) *ProfileConfig {
	if mode == "" {
		mode = self.Config.ProfileMode
	}
	if mode == "" || mode == DisableProfile || len(self.jobConfig.ProfileMode) == 0 {
		return nil
	}
	return self.jobConfig.ProfileMode[mode]
}

// FreeMemBytes returns the current amount of memory which the runtime may use
// for tasks like reading files.
//
// For the sake of consistency, if monitoring is enabled, this is 1GB.
// Otherwise, it will return 0 (unlimited).
func (self *Runtime) FreeMemBytes() int64 {
	return self.freeMemMB() * 1024 * 1024
}

// GetCallableFrom returns the named callable from the given include path.
func GetCallableFrom(pName, incPath string, mroPaths []string) (syntax.Callable, *syntax.TypeLookup, error) {
	if fpath, err := util.FindUniquePath(incPath, mroPaths); err != nil {
		return nil, nil, err
	} else if b, err := ioutil.ReadFile(fpath); err != nil {
		return nil, nil, err
	} else {
		var parser syntax.Parser
		if ast, err := parser.UncheckedParse(b, incPath); err != nil {
			return nil, nil, err
		} else {
			// Try to initialize the type table, but don't worry about
			// failures.  The includes were never parsed, so failures
			// are to be expected.
			_ = ast.CompileTypes()
			for _, c := range ast.Callables.List {
				if c.GetId() == pName {
					return c, &ast.TypeTable, nil
				}
			}
			return nil, &ast.TypeTable, &RuntimeError{
				Msg: fmt.Sprintf(
					"%q is not a declared pipeline or stage in %q",
					pName, fpath),
			}
		}
	}
}

// GetCallable searches every file in $MROPATH/[^_]*.mro until it finds one
// containing the given callable object (stage or pipeline) and returns it.
//
// If compile is true, the sources are "compiled" (checked for errors).
// Otherwise, keep in mind that some fields in the callable, such as the
// .Table fields of the parameter and binding lists, may not be fully
// populated.
func GetCallable(mroPaths []string, name string, compile bool) (syntax.Callable, *syntax.TypeLookup, error) {
	var parser syntax.Parser
	parse := parser.UncheckedParse
	if compile {
		parse = func(data []byte, fpath string) (*syntax.Ast, error) {
			_, _, ast, err := parser.ParseSourceBytes(
				data, fpath, mroPaths, true)
			return ast, err
		}
	}
	for _, mroPath := range mroPaths {
		if fpaths, err := util.Readdirnames(mroPath); err == nil {
			for _, fpath := range fpaths {
				if strings.HasPrefix(fpath, "_") || !strings.HasSuffix(fpath, ".mro") {
					// skip, private file
				} else if data, err := ioutil.ReadFile(path.Join(mroPath, fpath)); err == nil {
					if ast, err := parse(data, fpath); err == nil {
						for _, callable := range ast.Callables.List {
							if callable.GetId() == name {
								return callable, &ast.TypeTable, nil
							}
						}
					} else {
						return nil, nil, err
					}
				} else {
					return nil, nil, err
				}
			}
		} else {
			return nil, nil, err
		}
	}
	return nil, nil, &RuntimeError{
		Msg: fmt.Sprintf("'%s' is not a declared pipeline or stage", name),
	}
}

// possibleStructType returns true if the type name might refer to a struct
// type.  That is, if it is not an array, typed map, or builtin type.
func possibleStructType(tname syntax.TypeId, lookup *syntax.TypeLookup) bool {
	if tname.MapDim != 0 {
		// Maps are not structs.
		return false
	}
	if lookup == nil {
		// Neither is the builtin untyped map
		return tname.Tname != syntax.KindMap
	}
	t := lookup.Get(tname)
	if t == nil {
		return true
	}
	_, ok := t.(*syntax.StructType)
	return ok
}

// Recursively search an expression to convert MapExp to struct types where
// appropriate.  This should only get applied for expression types which are
// parsed from json, as opposed to those parsed from mro.
func fixExpressionTypes(exp syntax.Exp, tname syntax.TypeId, lookup *syntax.TypeLookup) {
	switch exp := exp.(type) {
	case *syntax.ArrayExp:
		if tname.ArrayDim > 0 {
			tname.ArrayDim--
		}
		for _, e := range exp.Value {
			fixExpressionTypes(e, tname, lookup)
		}
	case *syntax.MapExp:
		if tname.MapDim > 0 {
			tname.ArrayDim = tname.MapDim - 1
			tname.MapDim = 0
			for _, e := range exp.Value {
				fixExpressionTypes(e, tname, lookup)
			}
		} else if lookup == nil {
			if possibleStructType(tname, lookup) {
				exp.Kind = syntax.KindStruct
			}
		} else {
			t := lookup.Get(tname)
			if t != nil {
				if t, ok := t.(*syntax.StructType); ok {
					exp.Kind = syntax.KindStruct
					for _, member := range t.Members {
						fixExpressionTypes(exp.Value[member.Id], member.Tname, lookup)
					}
				}
			} else if possibleStructType(tname, lookup) {
				exp.Kind = syntax.KindStruct
			}
		}
	}
}

func convertToExp(parser *syntax.Parser, split bool, val json.Marshaler,
	tname syntax.TypeId, lookup *syntax.TypeLookup) (syntax.ValExp, error) {
	switch val := val.(type) {
	case syntax.ValExp:
		return val, nil
	case json.RawMessage:
		if split {
			var jv struct {
				Split json.RawMessage `json:"split"`
			}
			if err := json.Unmarshal(val, &jv); err != nil {
				return nil, err
			}
			exp, err := convertToExp(parser, false,
				jv.Split, tname, lookup)
			if n, ok := exp.(*syntax.NullExp); ok {
				return n, err
			}
			src, _ := exp.(syntax.MapCallSource)
			return &syntax.SplitExp{
				Value:  exp,
				Source: src,
			}, err
		}
		exp, err := parser.ParseValExp(val)
		fixExpressionTypes(exp, tname, lookup)
		return exp, err
	case LazyArgumentMap:
		res := syntax.MapExp{
			Kind:  syntax.KindMap,
			Value: make(map[string]syntax.Exp, len(val)),
		}
		if possibleStructType(tname, lookup) {
			res.Kind = syntax.KindStruct
		} else if tname.MapDim > 0 {
			tname.ArrayDim = tname.MapDim - 1
			tname.MapDim = 0
		}
		for k, v := range val {
			if e, err := convertToExp(parser, false,
				v, tname, lookup); err != nil {
				return &res, err
			} else {
				res.Value[k] = e
			}
		}
		return &res, nil
	case MarshalerMap:
		res := syntax.MapExp{
			Kind:  syntax.KindMap,
			Value: make(map[string]syntax.Exp, len(val)),
		}
		if possibleStructType(tname, lookup) {
			res.Kind = syntax.KindStruct
		} else if tname.MapDim > 0 {
			tname.ArrayDim = tname.MapDim - 1
			tname.MapDim = 0
		}
		for k, v := range val {
			if e, err := convertToExp(parser, false,
				v, tname, lookup); err != nil {
				return &res, err
			} else {
				res.Value[k] = e
			}
		}
		return &res, nil
	case marshallerArray:
		res := syntax.ArrayExp{
			Value: make([]syntax.Exp, 0, len(val)),
		}
		if tname.ArrayDim > 0 {
			tname.ArrayDim--
		}
		for _, v := range val {
			if e, err := convertToExp(parser, false,
				v, tname, lookup); err != nil {
				return &res, err
			} else {
				res.Value = append(res.Value, e)
			}
		}
		return &res, nil
	default:
		if val == nil {
			return new(syntax.NullExp), nil
		}
		// Simple types, e.g. string, boolean, number
		if b, err := val.MarshalJSON(); err != nil {
			return nil, err
		} else {
			return convertToExp(parser, split, json.RawMessage(b),
				tname, lookup)
		}
	}
}

func BuildCallAst(
	name string,
	args MarshalerMap,
	splitargs []string,
	callable syntax.Callable,
	lookup *syntax.TypeLookup,
	mroPaths []string) (*syntax.Ast, error) {
	ast := syntax.Ast{
		Call: &syntax.CallStm{
			Id:    name,
			DecId: callable.GetId(),
			Bindings: &syntax.BindStms{
				List: make([]*syntax.BindStm,
					0, len(callable.GetInParams().List)),
			},
		},
	}
	if f := callable.File(); f != nil && f.FileName != "" {
		rel, _, err := syntax.IncludeFilePath(f.FullPath, mroPaths)
		if (err != nil && rel == "") ||
			(filepath.IsAbs(rel) && !filepath.IsAbs(f.FileName)) {
			rel = f.FileName
		}
		ast.Includes = []*syntax.Include{{Value: rel}}
	}
	var parser syntax.Parser
	var null syntax.NullExp
	// for each parameter, either provide the value or null.
	for _, param := range callable.GetInParams().List {
		binding := syntax.BindStm{
			Id:    param.GetId(),
			Tname: param.GetTname(),
		}
		split := false
		for _, id := range splitargs {
			if id == param.GetId() {
				split = true
				break
			}
		}
		if val := args[param.GetId()]; val != nil {
			var err error
			binding.Exp, err = convertToExp(&parser, split, val, binding.Tname, lookup)
			s, ok := binding.Exp.(*syntax.SplitExp)
			if split && !ok {
				src, _ := binding.Exp.(syntax.MapCallSource)
				s = &syntax.SplitExp{
					Call:   ast.Call,
					Source: src,
					Value:  binding.Exp,
				}
				binding.Exp = s
			}
			if (split || ok) && ast.Call.Mapping == nil {
				ast.Call.Mapping = s.Source
			}

			if err != nil {
				return &ast, err
			}
		} else {
			binding.Exp = &null
		}
		ast.Call.Bindings.List = append(ast.Call.Bindings.List, &binding)
	}
	if len(splitargs) > 0 && ast.Call.Mapping == nil {
		// Will happen if uncompiled.  Add a placeholder.
		ast.Call.Mapping = new(syntax.NullExp)
	}
	return &ast, nil
}

// Deprecated: Use BuildCallAst instead.
func BuildCallSource(
	name string,
	args MarshalerMap,
	splitargs []string,
	callable syntax.Callable,
	lookup *syntax.TypeLookup,
	mroPaths []string) (string, error) {
	ast, err := BuildCallAst(name, args, splitargs, callable, lookup, mroPaths)
	if err != nil || ast == nil {
		return "", err
	}
	return ast.Format(), nil
}

func (invocation *InvocationData) BuildCallAst(mroPaths []string) (*syntax.Ast, error) {
	if invocation.Call == "" {
		return nil, fmt.Errorf("no pipeline or stage specified")
	}
	if len(invocation.SweepArgs) > 0 {
		return nil, fmt.Errorf("sweep is no longer supported - migrate to map call instead")
	}
	var callable syntax.Callable
	var lookup *syntax.TypeLookup
	if invocation.Include != "" {
		c, l, err := GetCallableFrom(
			invocation.Call, invocation.Include, mroPaths)
		if err != nil {
			return nil, err
		}
		callable = c
		lookup = l
	} else {
		c, l, err := GetCallable(mroPaths, invocation.Call, false)
		if err != nil {
			return nil, err
		}
		callable = c
		lookup = l
	}

	if invocation.Args == nil {
		return nil, fmt.Errorf("no args given")
	}

	return BuildCallAst(
		invocation.Call,
		invocation.Args.ToMarshalerMap(),
		invocation.SplitArgs,
		callable,
		lookup,
		mroPaths)
}

// Deprecated: Use BuildCallAst instead.
func (invocation *InvocationData) BuildCallSource(mroPaths []string) (string, error) {
	ast, err := invocation.BuildCallAst(mroPaths)
	if err != nil {
		return "", err
	}
	return ast.Format(), nil
}

// InvocationDataFromSource generates an InvocationData for which its
// BuildCallSource method will produce an equivalent call.
//
// That is, if one does
//
//   src, err := invocation1.BuildCallSource(mroPaths)
//   if err != nil {
//        panic(err)
//   }
//   invocation2, err := InvocationDataFromSource([]byte(src), mroPaths)
//
// then invocation2 should be equivalent to invocation1.
func InvocationDataFromSource(src []byte, mroPaths []string) (*InvocationData, error) {
	var parser syntax.Parser
	ast, err := parser.UncheckedParseIncludes(src, "", mroPaths)
	if err != nil {
		return nil, err
	}
	return BuildDataForAst(ast)
}

// Deprecated: Use InvocationDataFromSource instead.
func BuildCallData(src string, srcPath string, mroPaths []string) (*InvocationData, error) {
	_, _, ast, err := syntax.ParseSource(src, srcPath, mroPaths, false)
	if err != nil {
		return nil, err
	}
	return BuildDataForAst(ast)
}

func BuildDataForAst(ast *syntax.Ast) (*InvocationData, error) {
	if ast.Call == nil {
		return nil, &RuntimeError{
			Msg: "cannot jsonify a pipeline without a call statement",
		}
	}

	args := make(LazyArgumentMap, len(ast.Call.Bindings.List))
	var splitargs []string
	for _, binding := range ast.Call.Bindings.List {
		var err error
		args[binding.Id], err = binding.Exp.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("error serializing argument %s: %v",
				binding.Id, err)
		}
		if _, ok := binding.Exp.(*syntax.SplitExp); ok {
			splitargs = append(splitargs, binding.Id)
		}
	}
	var include string
	if ast.Callables != nil {
		if c := ast.Callables.Table[ast.Call.DecId]; c != nil {
			if f := c.File(); f != nil {
				include = f.FileName
			}
		} else {
			// Possibly not fully compiled, do a linear search instead.
			for _, c := range ast.Callables.List {
				if f := c.File(); c.GetId() == ast.Call.DecId && f != nil {
					include = f.FileName
					break
				}
			}
		}
	}
	if include == "" && len(ast.Includes) > 0 {
		// An ast where the includes were not parsed may not actually have a
		// callable definition.  Use the first available include and hope for
		// the best.
		if i := ast.Includes[0]; i != nil {
			include = i.Value
		}
	}
	return &InvocationData{
		Call:      ast.Call.DecId,
		Args:      args,
		SplitArgs: splitargs,
		Include:   include,
	}, nil
}
