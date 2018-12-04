// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime/trace"
	"strings"
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

func (self *RemoteJobManager) GetSystemReqs(threads int, memGB int) (int, int) {
	// Sanity check the thread count.
	if threads == 0 {
		threads = self.config.jobSettings.ThreadsPerJob
	} else if threads < 0 {
		threads = -threads
	}

	// Sanity check memory requirements.
	if memGB < 0 {
		// Negative request is a sentinel value requesting as much as possible.
		// For remote jobs, at least for now, give reserve the minimum usable.
		memGB = -memGB
	}
	if memGB < 1 {
		memGB = self.config.jobSettings.MemGBPerJob
	}

	// Compute threads needed based on memory requirements.
	if self.memGBPerCore > 0 {
		threads = max(threads, (memGB+self.memGBPerCore-1)/self.memGBPerCore)
	}

	// If threading is disabled, use only 1 thread.
	if !self.config.threadingEnabled {
		threads = 1
	}

	return threads, memGB
}

func (self *RemoteJobManager) execJob(shellCmd string, argv []string,
	envs map[string]string, metadata *Metadata, threads int, memGB int,
	special string, fqname string, shellName string, localpreflight bool) {
	ctx, task := trace.NewTask(context.Background(), "queueRemote")

	// no limit, send the job
	if self.maxJobs <= 0 {
		defer task.End()
		self.sendJob(shellCmd, argv, envs, metadata, threads, memGB, special, fqname, shellName, ctx)
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
		self.sendJob(shellCmd, argv, envs, metadata, threads, memGB, special, fqname, shellName, ctx)
	}()
}

func (self *RemoteJobManager) endJob(metadata *Metadata) {
	if self.jobSem != nil {
		self.jobSem.Release(metadata)
	}
}

func (self *RemoteJobManager) sendJob(shellCmd string, argv []string, envs map[string]string,
	metadata *Metadata, threads int, memGB int, special string, fqname string, shellName string,
	ctx context.Context) {

	if self.jobFreqMillis > 0 {
		<-(self.limiter.C)
		if self.debug {
			util.LogInfo("jobmngr", "Job rate-limit released: %s", fqname)
		}
	}
	threads, memGB = self.GetSystemReqs(threads, memGB)

	// figure out per-thread memory requirements for the template.  If
	// mempercore is specified, use that as what we send.
	memGBPerThread := memGB
	if self.memGBPerCore > 0 {
		memGBPerThread = self.memGBPerCore
	} else {
		// ceil to make sure that we're not starving a job
		memGBPerThread = memGB / threads
		if memGB%threads > 0 {
			memGBPerThread += 1
		}
	}

	mappedJobResourcesOpt := ""
	// If a __special is specified for this stage, and the runtime was called
	// with MRO_JOBRESOURCES defining a mapping from __special to a complex value
	// expression, then populate the resources option into the template. Otherwise,
	// leave it blank to revert to default behavior.
	if len(special) > 0 {
		if resources, ok := self.jobResourcesMappings[special]; ok {
			mappedJobResourcesOpt = strings.Replace(
				self.config.jobResourcesOpt,
				"__RESOURCES__", resources, 1)
		}
	}

	argv = append(
		util.FormatEnv(threadEnvs(self, threads, envs)),
		append([]string{shellCmd},
			argv...)...,
	)
	params := map[string]string{
		"JOB_NAME":          fqname + "." + shellName,
		"THREADS":           fmt.Sprintf("%d", threads),
		"STDOUT":            metadata.MetadataFilePath("stdout"),
		"STDERR":            metadata.MetadataFilePath("stderr"),
		"JOB_WORKDIR":       metadata.curFilesPath,
		"CMD":               strings.Join(argv, " "),
		"MEM_GB":            fmt.Sprintf("%d", memGB),
		"MEM_MB":            fmt.Sprintf("%d", memGB*1024),
		"MEM_KB":            fmt.Sprintf("%d", memGB*1024*1024),
		"MEM_B":             fmt.Sprintf("%d", memGB*1024*1024*1024),
		"MEM_GB_PER_THREAD": fmt.Sprintf("%d", memGBPerThread),
		"MEM_MB_PER_THREAD": fmt.Sprintf("%d", memGBPerThread*1024),
		"MEM_KB_PER_THREAD": fmt.Sprintf("%d", memGBPerThread*1024*1024),
		"MEM_B_PER_THREAD":  fmt.Sprintf("%d", memGBPerThread*1024*1024*1024),
		"ACCOUNT":           os.Getenv("MRO_ACCOUNT"),
		"RESOURCES":         mappedJobResourcesOpt,
	}

	// Replace template annotations with actual values
	args := []string{}
	template := self.config.jobTemplate
	for key, val := range params {
		if len(val) > 0 {
			args = append(args, fmt.Sprintf("__MRO_%s__", key), val)
		} else {
			// Remove line containing parameter from template
			for _, line := range strings.Split(template, "\n") {
				if strings.Contains(line, fmt.Sprintf("__MRO_%s__", key)) {
					template = strings.Replace(template, line, "", 1)
				}
			}
		}
	}
	r := strings.NewReplacer(args...)
	jobscript := r.Replace(template)
	metadata.WriteRaw("jobscript", jobscript)

	cmd := exec.CommandContext(ctx, self.config.jobCmd, self.config.jobCmdArgs...)
	cmd.Dir = metadata.curFilesPath
	cmd.Stdin = strings.NewReader(jobscript)

	util.EnterCriticalSection()
	defer util.ExitCriticalSection()
	metadata.remove("queued_locally")
	if output, err := cmd.CombinedOutput(); err != nil {
		metadata.WriteRaw(Errors, "jobcmd error ("+err.Error()+"):\n"+string(output))
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
