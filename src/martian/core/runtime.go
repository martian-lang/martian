// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

// Martian runtime. This contains the code to instantiate or restart
// pipestances.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"martian/syntax"
	"martian/util"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/satori/go.uuid"
)

const STAGE_TYPE_SPLIT = "split"
const STAGE_TYPE_CHUNK = "chunk"
const STAGE_TYPE_JOIN = "join"

const forkPrintInterval = 5 * time.Minute

// Helpers

func ParseFQName(fqname string) (string, string) {
	parts := strings.Split(fqname, ".")
	return parts[2], parts[1]
}

func MakeFQName(pipeline string, psid string) string {
	return fmt.Sprintf("ID.%s.%s", psid, pipeline)
}

func ParseTimestamp(data string) string {
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

func ParseJobMode(data string) (string, string, string) {
	jobmode := "local"
	if m := regexp.MustCompile(".*--jobmode=([^\\s]+).*").FindStringSubmatch(data); len(m) > 0 {
		jobmode = m[1]
	}
	localcores := "max"
	if m := regexp.MustCompile(".*--localcores=([^\\s]+).*").FindStringSubmatch(data); len(m) > 0 {
		localcores = m[1]
	}
	localmem := "max"
	if m := regexp.MustCompile(".*--localmem=([^\\s]+).*").FindStringSubmatch(data); len(m) > 0 {
		localmem = m[1]
	}
	return jobmode, localcores, localmem
}

func VerifyVDRMode(vdrMode string) {
	validModes := []string{"rolling", "post", "disable"}
	for _, validMode := range validModes {
		if validMode == vdrMode {
			return
		}
	}
	util.PrintInfo("runtime", "Invalid VDR mode: %s. Valid VDR modes: %s", vdrMode, strings.Join(validModes, ", "))
	os.Exit(1)
}

func VerifyOnFinish(onfinish string) {
	if _, err := exec.LookPath(onfinish); err != nil {
		util.PrintInfo("runtime", "Invalid onfinish hook executable (%v): %v", err, onfinish)
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

// Configuration required to initialize a Runtime object.
type RuntimeOptions struct {
	// The runtime mode (required): either "local" or a named mode from
	// jobmanagers/config.json
	JobMode string

	// The volatile disk recovery mode (required): either "post",
	// "rolling", or "disable".
	VdrMode string

	// The profiling mode (required): "disable" or one of the available
	// constants.
	ProfileMode     ProfileMode
	MartianVersion  string
	LocalMem        int
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
}

// Collects configuration and state required to initialize and run pipestances
// and stagestances.
type Runtime struct {
	adaptersPath    string
	mrjob           string
	martianVersion  string
	vdrMode         string
	jobMode         string
	profileMode     ProfileMode
	MroCache        *MroCache
	JobManager      JobManager
	LocalJobManager JobManager
	fullStageReset  bool
	enableStackVars bool
	enableZip       bool
	skipPreflight   bool
	enableMonitor   bool
	stest           bool
	onFinishExec    string
	overrides       *PipestanceOverrides
}

// Deprecated: use RuntimeConfig.NewRuntime() instead
func NewRuntime(jobMode string, vdrMode string, profileMode ProfileMode, martianVersion string) *Runtime {
	return NewRuntimeWithCores(jobMode, vdrMode, profileMode, martianVersion,
		-1, -1, -1, -1, -1, "", false, false, false, false, false, false, false, "", nil, false)
}

// Deprecated: use RuntimeConfig.NewRuntime() instead
func NewRuntimeWithCores(jobMode string, vdrMode string, profileMode ProfileMode, martianVersion string,
	reqCores int, reqMem int, reqMemPerCore int, maxJobs int, jobFreqMillis int, jobQueues string,
	fullStageReset bool, enableStackVars bool, enableZip bool, skipPreflight bool, enableMonitor bool,
	debug bool, stest bool, onFinishExec string, overrides *PipestanceOverrides, limitLoadavg bool) *Runtime {
	c := RuntimeOptions{
		JobMode:         jobMode,
		VdrMode:         vdrMode,
		ProfileMode:     profileMode,
		MartianVersion:  martianVersion,
		LocalMem:        reqMem,
		LocalCores:      reqCores,
		MemPerCore:      reqMemPerCore,
		MaxJobs:         maxJobs,
		JobFreqMillis:   jobFreqMillis,
		ResourceSpecial: jobQueues,
		FullStageReset:  fullStageReset,
		StackVars:       enableStackVars,
		Zip:             enableZip,
		SkipPreflight:   skipPreflight,
		Monitor:         enableMonitor,
		Debug:           debug,
		StressTest:      stest,
		OnFinishHandler: onFinishExec,
		Overrides:       overrides,
		LimitLoadavg:    limitLoadavg,
	}
	return c.NewRuntime()
}

func DefaultRuntimeOptions() RuntimeOptions {
	return RuntimeOptions{
		MartianVersion: util.GetVersion(),
		ProfileMode:    DisableProfile,
		JobMode:        "local",
		VdrMode:        "post",
	}
}

func (c *RuntimeOptions) NewRuntime() *Runtime {
	self := &Runtime{}
	self.adaptersPath = util.RelPath(path.Join("..", "adapters"))
	self.mrjob = util.RelPath("mrjob")
	self.martianVersion = c.MartianVersion
	self.jobMode = c.JobMode
	self.vdrMode = c.VdrMode
	self.profileMode = c.ProfileMode
	self.fullStageReset = c.FullStageReset
	self.enableStackVars = c.StackVars
	self.enableZip = c.Zip
	self.skipPreflight = c.SkipPreflight
	self.enableMonitor = c.Monitor
	self.stest = c.StressTest
	self.onFinishExec = c.OnFinishHandler

	self.MroCache = NewMroCache()
	self.LocalJobManager = NewLocalJobManager(c.LocalCores, c.LocalMem, c.Debug,
		c.LimitLoadavg,
		self.jobMode != "local")
	if self.jobMode == "local" {
		self.JobManager = self.LocalJobManager
	} else {
		self.JobManager = NewRemoteJobManager(self.jobMode, c.MemPerCore, c.MaxJobs,
			c.JobFreqMillis, c.ResourceSpecial, c.Debug)
	}
	VerifyVDRMode(self.vdrMode)
	VerifyProfileMode(self.profileMode)

	if c.Overrides == nil {
		self.overrides, _ = ReadOverrides("")
	} else {
		self.overrides = c.Overrides
	}

	return self
}

// Compile all the MRO files in mroPaths.
func (self *Runtime) CompileAll(mroPaths []string, checkSrcPath bool) (int, []*syntax.Ast, error) {
	numFiles := 0
	asts := []*syntax.Ast{}
	for _, mroPath := range mroPaths {
		fpaths, _ := filepath.Glob(mroPath + "/[^_]*.mro")
		for _, fpath := range fpaths {
			if _, _, ast, err := syntax.Compile(fpath, mroPaths, checkSrcPath); err != nil {
				return 0, []*syntax.Ast{}, err
			} else {
				asts = append(asts, ast)
			}
		}
		numFiles += len(fpaths)
	}
	return numFiles, asts, nil
}

// Instantiate a pipestance object given a psid, MRO source, and a
// pipestance path. This is the core (private) method called by the
// public InvokeWithSource and Reattach methods.
func (self *Runtime) instantiatePipeline(src string, srcPath string, psid string,
	pipestancePath string, mroPaths []string, mroVersion string,
	envs map[string]string, readOnly bool) (string, *Pipestance, error) {
	// Parse the invocation source.
	postsrc, _, ast, err := syntax.ParseSource(src, srcPath, mroPaths, !readOnly)
	if err != nil {
		return "", nil, err
	}

	// Check there's a call.
	if ast.Call == nil {
		return "", nil, &RuntimeError{"cannot start a pipeline without a call statement"}
	}
	// Make sure it's a pipeline we're calling.
	if pipeline := ast.Callables.Table[ast.Call.Id]; pipeline == nil {
		return "", nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared pipeline", ast.Call.Id)}
	}

	invocationData, _ := BuildCallData(src, srcPath, mroPaths)

	// Instantiate the pipeline.
	if err := CheckMinimalSpace(pipestancePath); err != nil {
		return "", nil, err
	}
	pipestance, err := NewPipestance(NewTopNode(self, psid, pipestancePath, mroPaths, mroVersion, envs, invocationData),
		ast.Call, ast.Callables)
	if err != nil {
		return "", nil, err
	}

	// Lock the pipestance if not in read-only mode.
	if !readOnly {
		if err := pipestance.Lock(); err != nil {
			return "", nil, err
		}
	}

	pipestance.getNode().mkdirs()

	return postsrc, pipestance, nil
}

// Invokes a new pipestance.
func (self *Runtime) InvokePipeline(src string, srcPath string, psid string,
	pipestancePath string, mroPaths []string, mroVersion string,
	envs map[string]string, tags []string) (*Pipestance, error) {

	// Error if pipestance directory is non-empty, otherwise create.
	if _, err := os.Stat(pipestancePath); err == nil {
		if fileNames, err := util.Readdirnames(pipestancePath); err != nil || len(fileNames) > 0 {
			return nil, &PipestanceExistsError{psid}
		}
	} else if err := os.MkdirAll(pipestancePath, 0777); err != nil {
		return nil, err
	}

	// Expand env vars in invocation source and instantiate.
	src = os.ExpandEnv(src)
	readOnly := false
	postsrc, pipestance, err := self.instantiatePipeline(src, srcPath, psid, pipestancePath, mroPaths,
		mroVersion, envs, readOnly)
	if err != nil {
		// If instantiation failed, delete the pipestance folder.
		os.RemoveAll(pipestancePath)
		return nil, err
	}

	// Write top-level metadata files.
	pipestance.metadata.WriteRaw(InvocationFile, src)
	pipestance.metadata.WriteRaw(JobModeFile, self.jobMode)
	pipestance.metadata.WriteRaw(MroSourceFile, postsrc)
	pipestance.metadata.Write(VersionsFile, &VersionInfo{
		Martian:   self.martianVersion,
		Pipelines: mroVersion,
	})
	pipestance.metadata.Write(TagsFile, tags)
	if uid := os.Getenv("MRO_FORCE_UUID"); uid == "" {
		pipestance.SetUuid(uuid.NewV4().String())
	} else {
		util.LogInfo("runtime", "UUID forced to %s by environment", uid)
		pipestance.SetUuid(uid)
	}
	pipestance.metadata.WriteRaw(TimestampFile, "start: "+util.Timestamp())

	return pipestance, nil
}

func (self *Runtime) ReattachToPipestance(psid string, pipestancePath string, src string, mroPaths []string,
	mroVersion string, envs map[string]string, checkSrc bool, readOnly bool) (*Pipestance, error) {
	return self.reattachToPipestance(psid, pipestancePath, src, mroPaths, mroVersion, envs, checkSrc,
		readOnly, "invocation")
}

func (self *Runtime) ReattachToPipestanceWithMroSrc(psid string, pipestancePath string, src string, mroPaths []string,
	mroVersion string, envs map[string]string, checkSrc bool, readOnly bool) (*Pipestance, error) {
	return self.reattachToPipestance(psid, pipestancePath, src, mroPaths, mroVersion, envs, checkSrc,
		readOnly, "mrosource")
}

// Reattaches to an existing pipestance.
func (self *Runtime) reattachToPipestance(psid string, pipestancePath string, src string, mroPaths []string,
	mroVersion string, envs map[string]string, checkSrc bool, readOnly bool,
	srcType string) (*Pipestance, error) {
	fname := "_" + srcType
	invocationPath := path.Join(pipestancePath, fname)
	metadataPath := path.Join(pipestancePath, "_metadata.zip")

	// Read in the existing _invocation file.
	data, err := ioutil.ReadFile(invocationPath)
	if err != nil {
		return nil, &PipestancePathError{pipestancePath}
	}

	// Check if _invocation has changed.
	if checkSrc && src != string(data) {
		return nil, &PipestanceInvocationError{psid, invocationPath}
	}

	// Instantiate the pipestance.
	_, pipestance, err := self.instantiatePipeline(string(data), invocationPath, psid, pipestancePath, mroPaths,
		mroVersion, envs, readOnly)
	if err != nil {
		return nil, err
	}

	// If _jobmode exists, make sure we reattach to pipestance in the same job mode.
	if !readOnly {
		if err := pipestance.VerifyJobMode(); err != nil {
			pipestance.Unlock()
			return nil, err
		}
	}

	// If _metadata exists, unzip it so the pipestance can reads its metadata.
	if _, err := os.Stat(metadataPath); err == nil {
		if err := util.Unzip(metadataPath); err != nil {
			pipestance.Unlock()
			return nil, err
		}
		os.Remove(metadataPath)
	}

	// If we're reattaching in local mode, restart any stages that were
	// left in a running state from last mrp run. The actual job would
	// have been killed by the CTRL-C.
	if !readOnly {
		util.PrintInfo("runtime", "Reattaching in %s mode.", self.jobMode)
		if err = pipestance.RestartRunningNodes(self.jobMode); err != nil {
			pipestance.Unlock()
			return nil, err
		}
	}

	return pipestance, nil
}

// Instantiate a stagestance.
func (self *Runtime) InvokeStage(src string, srcPath string, ssid string,
	stagestancePath string, mroPaths []string, mroVersion string,
	envs map[string]string) (*Stagestance, error) {
	// Check if stagestance path already exists.
	if _, err := os.Stat(stagestancePath); err == nil {
		return nil, &RuntimeError{fmt.Sprintf("stagestance '%s' already exists", ssid)}
	} else if err := os.MkdirAll(stagestancePath, 0777); err != nil {
		return nil, err
	}

	// Parse the invocation source.
	src = os.ExpandEnv(src)
	_, _, ast, err := syntax.ParseSource(src, srcPath, mroPaths, true)
	if err != nil {
		return nil, err
	}

	// Check there's a call.
	if ast.Call == nil {
		return nil, &RuntimeError{"cannot start a stage without a call statement"}
	}
	// Make sure it's a stage we're calling.
	if _, ok := ast.Callables.Table[ast.Call.Id].(*syntax.Stage); !ok {
		return nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared stage", ast.Call.Id)}
	}

	invocationData, _ := BuildCallData(src, srcPath, mroPaths)

	// Instantiate stagestance.
	stagestance, err := NewStagestance(NewTopNode(self, "", stagestancePath, mroPaths, mroVersion, envs, invocationData),
		ast.Call, ast.Callables)
	if err != nil {
		return nil, err
	}

	stagestance.getNode().mkdirs()

	return stagestance, nil
}

func (self *Runtime) GetSerializationInto(pipestancePath string, name MetadataFileName, target interface{}) error {
	metadata := NewMetadata("", pipestancePath)
	return metadata.ReadInto(name, target)
}

func (self *Runtime) GetSerialization(pipestancePath string, name MetadataFileName) (interface{}, bool) {
	metadata := NewMetadata("", pipestancePath)
	metadata.loadCache()
	if metadata.exists(name) {
		return metadata.read(name), true
	}
	return nil, false
}

func (self *Runtime) GetMetadata(pipestancePath string, metadataPath string) (string, error) {
	metadata := NewMetadata("", pipestancePath)
	metadata.loadCache()
	if metadata.exists(MetadataZip) {
		relPath, _ := filepath.Rel(pipestancePath, metadataPath)

		// Relative paths outside the pipestance directory will be ignored.
		if !strings.Contains(relPath, "..") {
			if data, err := util.ReadZip(metadata.MetadataFilePath(MetadataZip), relPath); err == nil {
				return data, nil
			}
		}
	}
	data, err := ioutil.ReadFile(metadataPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type MroCache struct {
	callableTable map[string]map[string]syntax.Callable
	pipelines     map[string]bool
}

func NewMroCache() *MroCache {
	self := &MroCache{}
	self.callableTable = map[string]map[string]syntax.Callable{}
	self.pipelines = map[string]bool{}

	return self
}

func (self *MroCache) CacheMros(mroPaths []string) {
	for _, mroPath := range mroPaths {
		self.callableTable[mroPath] = map[string]syntax.Callable{}
		fpaths, _ := filepath.Glob(mroPath + "/[^_]*.mro")
		for _, fpath := range fpaths {
			if data, err := ioutil.ReadFile(fpath); err == nil {
				if _, _, ast, err := syntax.ParseSource(string(data), fpath, mroPaths, true); err == nil {
					for _, callable := range ast.Callables.Table {
						self.callableTable[mroPath][callable.GetId()] = callable
						if _, ok := callable.(*syntax.Pipeline); ok {
							self.pipelines[callable.GetId()] = true
						}
					}
				}
			}
		}
	}
}

func (self *MroCache) GetPipelines() []string {
	pipelines := []string{}
	for pipeline := range self.pipelines {
		pipelines = append(pipelines, pipeline)
	}
	return pipelines
}

func (self *MroCache) GetCallable(mroPaths []string, name string) (syntax.Callable, error) {
	for _, mroPath := range mroPaths {
		// Make sure MROs from mroPath have been loaded.
		if _, ok := self.callableTable[mroPath]; !ok {
			return nil, &RuntimeError{fmt.Sprintf("MROs from mro path '%s' have not been loaded", mroPath)}
		}

		// Make sure pipeline has been loaded
		if callable, ok := self.callableTable[mroPath][name]; ok {
			return callable, nil
		}
	}
	return nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared pipeline or stage", name)}
}

func GetCallable(mroPaths []string, name string) (syntax.Callable, error) {
	for _, mroPath := range mroPaths {
		if fpaths, err := filepath.Glob(mroPath + "/[^_]*.mro"); err == nil {
			for _, fpath := range fpaths {
				if data, err := ioutil.ReadFile(fpath); err == nil {
					if _, _, ast, err := syntax.ParseSource(
						string(data), fpath, mroPaths, true); err == nil {
						for _, callable := range ast.Callables.Table {
							if callable.GetId() == name {
								return callable, nil
							}
						}
					} else {
						return nil, err
					}
				} else {
					return nil, err
				}
			}
		} else {
			return nil, err
		}
	}
	return nil, &RuntimeError{fmt.Sprintf("'%s' is not a declared pipeline or stage", name)}
}

func buildVal(param syntax.Param, val interface{}) string {
	indent := "    "
	if data, err := json.MarshalIndent(val, "", indent); err == nil {
		// Indent multi-line values (but not first line).
		sublines := strings.Split(string(data), "\n")
		for i := range sublines[1:] {
			sublines[i+1] = indent + sublines[i+1]
		}
		return strings.Join(sublines, "\n")
	}
	return fmt.Sprintf("<ParseError: %v>", val)
}

func (self *Runtime) BuildCallSource(incpaths []string, name string, args map[string]interface{},
	sweepargs []string, mroPaths []string) (string, error) {
	callable, err := self.MroCache.GetCallable(mroPaths, name)
	if err != nil {
		util.LogInfo("package", "Could not get callable: %s", name)
		return "", err
	}
	return BuildCallSource(incpaths, name, args, sweepargs, callable)
}

func BuildCallSource(incpaths []string,
	name string,
	args map[string]interface{},
	sweepargs []string,
	callable syntax.Callable) (string, error) {
	// Build @include statements.
	includes := []string{}
	for _, incpath := range incpaths {
		includes = append(includes, fmt.Sprintf("@include \"%s\"", incpath))
	}
	// Loop over the pipeline's in params and print a binding
	// whether the args bag has a value for it not.
	lines := []string{}
	for _, param := range callable.GetInParams().List {
		valstr := buildVal(param, args[param.GetId()])

		for _, id := range sweepargs {
			if id == param.GetId() {
				valstr = fmt.Sprintf("sweep(%s)", strings.Trim(valstr, "[]"))
				break
			}
		}

		lines = append(lines, fmt.Sprintf("    %s = %s,", param.GetId(), valstr))
	}
	return fmt.Sprintf("%s\n\ncall %s(\n%s\n)", strings.Join(includes, "\n"),
		name, strings.Join(lines, "\n")), nil
}

func BuildCallData(src string, srcPath string, mroPaths []string) (*InvocationData, error) {
	_, incpaths, ast, err := syntax.ParseSource(src, srcPath, mroPaths, false)
	if err != nil {
		return nil, err
	}

	if ast.Call == nil {
		return nil, &RuntimeError{"cannot jsonify a pipeline without a call statement"}
	}

	args := map[string]interface{}{}
	sweepargs := []string{}
	for _, binding := range ast.Call.Bindings.List {
		args[binding.Id] = binding.Exp.ToInterface()
		if binding.Sweep {
			sweepargs = append(sweepargs, binding.Id)
		}
	}
	return &InvocationData{
		Call:         ast.Call.Id,
		Args:         args,
		SweepArgs:    sweepargs,
		IncludePaths: incpaths,
	}, nil
}
