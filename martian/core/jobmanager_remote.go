// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package core

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

type RemoteJobManager struct {
	jobMode              string
	jobResourcesMappings map[string]string
	config               jobManagerConfig
	memGBPerCore         int
	maxJobs              int
	jobFreqMillis        int
	jobSem               *MaxJobsSemaphore
	limiter              *time.Ticker
	debug                bool
	queueMutex           sync.Mutex
}

func NewRemoteJobManager(jobMode string, memGBPerCore int, maxJobs int, jobFreqMillis int,
	jobResources string, config *JobManagerJson, debug bool) *RemoteJobManager {
	self := &RemoteJobManager{}
	self.jobMode = jobMode
	self.memGBPerCore = memGBPerCore
	self.maxJobs = maxJobs
	self.jobFreqMillis = jobFreqMillis
	self.debug = debug
	self.config = verifyJobManager(jobMode, config, memGBPerCore)

	// Parse jobresources mappings
	self.jobResourcesMappings = map[string]string{}
	for _, mapping := range strings.Split(jobResources, ";") {
		if len(mapping) > 0 {
			parts := strings.Split(mapping, ":")
			if len(parts) == 2 {
				self.jobResourcesMappings[parts[0]] = parts[1]
				util.LogInfo("jobmngr", "Mapping %s to %s", parts[0], parts[1])
			} else {
				util.LogInfo("jobmngr", "Could not parse mapping: %s", mapping)
			}
		}
	}

	if self.maxJobs > 0 {
		self.jobSem = NewMaxJobsSemaphore(self.maxJobs)
	}
	if self.jobFreqMillis > 0 {
		self.limiter = time.NewTicker(time.Millisecond * time.Duration(self.jobFreqMillis))
	} else {
		// dummy limiter to keep struct OK
		self.limiter = time.NewTicker(time.Millisecond * 1)
	}
	return self
}

func (self *RemoteJobManager) refreshResources(bool) error {
	if self.jobSem != nil {
		self.jobSem.FindDone()
	}
	return nil
}

func (self *RemoteJobManager) GetMaxCores() int {
	return 0
}

func (self *RemoteJobManager) GetMaxMemGB() int {
	return 0
}

func (self *RemoteJobManager) GetSettings() *JobManagerSettings {
	return self.config.jobSettings
}

func (self *RemoteJobManager) GetSystemReqs(resRequest *JobResources) JobResources {
	res := *resRequest
	// Sanity check the thread count.
	if res.Threads == 0 {
		res.Threads = self.config.jobSettings.ThreadsPerJob
	} else if res.Threads < 0 {
		res.Threads = -res.Threads
	}

	// Sanity check memory requirements.
	if res.MemGB < 0 {
		// Negative request is a sentinel value requesting as much as possible.
		// For remote jobs, at least for now, give reserve the minimum usable.
		res.MemGB = -res.MemGB
	}
	if res.MemGB < 1 {
		res.MemGB = self.config.jobSettings.MemGBPerJob
	}
	if res.VMemGB < 1 {
		res.VMemGB = res.MemGB + self.config.jobSettings.ExtraVmemGB
	}

	// Compute threads needed based on memory requirements.
	if self.memGBPerCore > 0 {
		res.Threads = max(res.Threads, (res.MemGB+self.memGBPerCore-1)/self.memGBPerCore)
	}

	// If threading is disabled, use only 1 thread.
	if !self.config.threadingEnabled {
		res.Threads = 1
	}

	return res
}

func (self *RemoteJobManager) execJob(shellCmd string, argv []string,
	envs map[string]string, metadata *Metadata, resRequest *JobResources,
	fqname string, shellName string, localpreflight bool) {
	ctx, task := trace.NewTask(context.Background(), "queueRemote")

	// no limit, send the job
	if self.maxJobs <= 0 {
		defer task.End()
		self.sendJob(shellCmd, argv, envs,
			metadata, resRequest,
			fqname, shellName, ctx)
		return
	}

	// grab job when ready, block until job state changes to a finalized state
	go func() {
		defer task.End()
		if self.debug {
			util.LogInfo("jobmngr", "Waiting for job: %s", fqname)
		}
		// if we want to try to put a more precise cap on cluster execution load,
		// might be preferable to request num threads here instead of a slot per job
		if success := self.jobSem.Acquire(metadata); !success {
			return
		}
		if self.debug {
			util.LogInfo("jobmngr", "Job sent: %s", fqname)
		}
		self.sendJob(shellCmd, argv, envs,
			metadata, resRequest,
			fqname, shellName, ctx)
	}()
}

func (self *RemoteJobManager) endJob(metadata *Metadata) {
	if self.jobSem != nil {
		self.jobSem.Release(metadata)
	}
}

func (self *RemoteJobManager) jobScript(
	shellCmd string, argv []string, envs map[string]string,
	metadata *Metadata,
	resRequest *JobResources,
	fqname, shellName string) string {

	res := self.GetSystemReqs(resRequest)

	// figure out per-thread memory requirements for the template.
	// ceil to make sure that we're not starving a job.
	vmemGBPerThread := (res.VMemGB + res.Threads - 1) / res.Threads
	if self.memGBPerCore > vmemGBPerThread {
		vmemGBPerThread = self.memGBPerCore
	}
	memGBPerThread := vmemGBPerThread
	if self.config.alwaysVmem && res.VMemGB > res.MemGB {
		res.MemGB = res.VMemGB
	} else {
		memGBPerThread = (res.MemGB + res.Threads - 1) / res.Threads
		if self.memGBPerCore > memGBPerThread {
			memGBPerThread = self.memGBPerCore
		}
	}

	mappedJobResourcesOpt := ""
	// If a __special is specified for this stage, and the runtime was called
	// with MRO_JOBRESOURCES defining a mapping from __special to a complex value
	// expression, then populate the resources option into the template. Otherwise,
	// leave it blank to revert to default behavior.
	if len(res.Special) > 0 {
		if resources, ok := self.jobResourcesMappings[res.Special]; ok {
			mappedJobResourcesOpt = strings.Replace(
				self.config.jobResourcesOpt,
				"__RESOURCES__", resources, 1)
		}
	}

	argv = append(
		util.FormatEnv(threadEnvs(self, res.Threads, envs)),
		append([]string{shellCmd},
			argv...)...,
	)
	params := map[string]string{
		"JOB_NAME":           fqname + "." + shellName,
		"THREADS":            strconv.Itoa(res.Threads),
		"STDOUT":             metadata.MetadataFilePath("stdout"),
		"STDERR":             metadata.MetadataFilePath("stderr"),
		"JOB_WORKDIR":        metadata.curFilesPath,
		"CMD":                strings.Join(argv, " \\\n  "),
		"MEM_GB":             strconv.Itoa(res.MemGB),
		"MEM_MB":             strconv.Itoa(res.MemGB * 1024),
		"MEM_KB":             strconv.Itoa(res.MemGB * 1024 * 1024),
		"MEM_B":              strconv.Itoa(res.MemGB * 1024 * 1024 * 1024),
		"MEM_GB_PER_THREAD":  strconv.Itoa(memGBPerThread),
		"MEM_MB_PER_THREAD":  strconv.Itoa(memGBPerThread * 1024),
		"MEM_KB_PER_THREAD":  strconv.Itoa(memGBPerThread * 1024 * 1024),
		"MEM_B_PER_THREAD":   strconv.Itoa(memGBPerThread * 1024 * 1024 * 1024),
		"VMEM_GB":            strconv.Itoa(res.VMemGB),
		"VMEM_MB":            strconv.Itoa(res.VMemGB * 1024),
		"VMEM_KB":            strconv.Itoa(res.VMemGB * 1024 * 1024),
		"VMEM_B":             strconv.Itoa(res.VMemGB * 1024 * 1024 * 1024),
		"VMEM_GB_PER_THREAD": strconv.Itoa(vmemGBPerThread),
		"VMEM_MB_PER_THREAD": strconv.Itoa(vmemGBPerThread * 1024),
		"VMEM_KB_PER_THREAD": strconv.Itoa(vmemGBPerThread * 1024 * 1024),
		"VMEM_B_PER_THREAD":  strconv.Itoa(vmemGBPerThread * 1024 * 1024 * 1024),
		"ACCOUNT":            os.Getenv("MRO_ACCOUNT"),
		"RESOURCES":          mappedJobResourcesOpt,
	}

	template := self.config.jobTemplate
	// Replace template annotations with actual values
	args := make([]string, 0, 2*len(params))
	for key, val := range params {
		rkey := "__MRO_" + key + "__"
		if len(val) > 0 {
			args = append(args, rkey, val)
		} else {
			// Remove lines containing parameter from template
			for _, line := range strings.Split(template, "\n") {
				if strings.Contains(line, rkey) {
					args = append(args, line, "")
				}
			}
		}
	}
	r := strings.NewReplacer(args...)
	return r.Replace(template)
}

func (self *RemoteJobManager) sendJob(shellCmd string, argv []string, envs map[string]string,
	metadata *Metadata, resRequest *JobResources, fqname string, shellName string,
	ctx context.Context) {
	jobscript := self.jobScript(shellCmd, argv, envs, metadata,
		resRequest, fqname, shellName)
	metadata.WriteRaw("jobscript", jobscript)

	cmd := exec.CommandContext(ctx, self.config.jobCmd, self.config.jobCmdArgs...)
	cmd.Dir = metadata.curFilesPath
	cmd.Stdin = strings.NewReader(jobscript)

	// Regardless of the limiter rate, only allow one pending submission to the queue
	// at a time.  Otherwise there's a risk that if the submit command takes longer
	// than jobFreqMillis, commands will still pile up.  It's also a more "natural"
	// way to limit the submit rate if the submit server can't keep up.
	self.queueMutex.Lock()
	defer self.queueMutex.Unlock()
	if self.jobFreqMillis > 0 {
		<-(self.limiter.C)
		if self.debug {
			util.LogInfo("jobmngr", "Job rate-limit released: %s", fqname)
		}
	}

	util.EnterCriticalSection()
	defer util.ExitCriticalSection()
	if err := metadata.remove("queued_locally"); err != nil {
		util.LogError(err, "jobmngr", "Error removing queue sentinel file.")
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		metadata.WriteErrorString(
			"jobcmd error (" + err.Error() + "):\n" + string(output))
	} else {
		trimmed := bytes.TrimSpace(output)
		// jobids should not have spaces in them.  This is the most general way to
		// check that a string is actually a jobid.
		if len(trimmed) > 0 && !bytes.ContainsAny(trimmed, " \t\n\r") {
			metadata.WriteRawBytes("jobid", bytes.TrimSpace(output))
			metadata.cache("jobid", metadata.uniquifier)
		}
	}
}

func (self *RemoteJobManager) checkQueue(ids []string, ctx context.Context) ([]string, string) {
	if self.config.queueQueryCmd == "" {
		return ids, ""
	}
	jobPath := util.RelPath(path.Join("..", "jobmanagers"))
	cmd := exec.CommandContext(ctx, path.Join(jobPath, self.config.queueQueryCmd))
	cmd.Dir = jobPath
	cmd.Stdin = strings.NewReader(strings.Join(ids, "\n"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return ids, stderr.String()
	}
	return strings.Split(string(output), "\n"), stderr.String()
}

func (self *RemoteJobManager) hasQueueCheck() bool {
	return self.config.queueQueryCmd != ""
}

func (self *RemoteJobManager) queueCheckGrace() time.Duration {
	return self.config.queueQueryGrace
}
