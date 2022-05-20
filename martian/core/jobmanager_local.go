// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package core

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"runtime"
	"runtime/trace"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/martian-lang/martian/martian/util"
)

const (
	// These constants deal with a bug in the linux kernel used by certain
	// versions of RHEL 5 and 6, specifically
	// https://access.redhat.com/solutions/53258
	maxRetries     = 5
	exitCodeString = "errno 513"
)

const (
	// The following constants are used to estimate process (thread) counts.
	// In local mode, these are used to estimate the number of processes spawned,
	// to avoid bumping into the process ulimit.  This is all very rough approximation.

	// The approximate number of physical threads assumed used by the user at
	// startup, including those used by mrp.  The actual value is used as well in
	// real time, but this amount is reserved in the semaphore regardless.
	startingThreadCount = 45

	// The base number of threads assumed per job.  This includes the threads used
	// by the mrjob process and whatever threads the spawned process uses for
	// runtime management.  Very approximate since it depends on many details of
	// the stage code language and implementation.  The number of threads reserved
	// by the job will be added to this number.
	procsPerJob = 15
)

type LocalJobManager struct {
	maxCores    int
	maxMemGB    int
	maxVmemMB   int64
	jobSettings *JobManagerSettings
	centcoreSem *ResourceSemaphore
	memMBSem    *ResourceSemaphore
	vmemMBSem   *ResourceSemaphore
	procsSem    *ResourceSemaphore
	lastMemDiff int64
	queue       []*exec.Cmd
	debug       bool
	limitLoad   bool
	highMem     ObservedMemory
	jobDone     chan struct{}
}

func NewLocalJobManager(userMaxCores int,
	userMaxMemGB, userMaxVMemGB int,
	debug bool, limitLoadavg bool, clusterMode bool,
	config *JobManagerJson) *LocalJobManager {
	self := &LocalJobManager{
		debug:     debug,
		limitLoad: limitLoadavg,

		// Buffer up to 1 notification, in case a job finishes while the
		// runloop processing is in progress.
		jobDone: make(chan struct{}, 1),
	}
	self.jobSettings = verifyJobManager("local", config, -1).jobSettings
	self.setMaxCores(userMaxCores, clusterMode)
	self.setMaxMem(userMaxMemGB, userMaxVMemGB, clusterMode)
	self.setupSemaphores()
	return self
}

func (self *LocalJobManager) setMaxCores(userMaxCores int, clusterMode bool) {
	// Set Max number of cores usable at one time.
	if userMaxCores > 0 {
		// If user specified --localcores, use that value for Max usable cores.
		self.maxCores = userMaxCores
		util.LogInfo("jobmngr", "Using %d core%s, per --localcores option.",
			self.maxCores, util.Pluralize(self.maxCores))
	} else {
		if clusterMode {
			self.maxCores = self.jobSettings.ThreadsPerJob
		} else {
			// Otherwise, set Max usable cores to total number of cores reported
			// by the system.
			self.maxCores = runtime.NumCPU()
			util.LogInfo("jobmngr", "Using %d logical core%s available on system.",
				self.maxCores, util.Pluralize(self.maxCores))
		}
	}
}

func (self *LocalJobManager) setMaxMem(userMaxMemGB, userMaxVMemGB int, clusterMode bool) {
	var sysMem MemInfo
	if err := sysMem.Get(); err != nil && sysMem.Total == 0 {
		util.PrintError(err, "jobmngr",
			"Error attempting to read system memory values.")
	}
	cgMem, cgSoftLimit, cgUse := util.GetCgroupMemoryLimit()

	// Set Max GB of memory usable at one time.
	if userMaxMemGB > 0 {
		// If user specified --localmem, use that value for Max usable GB.
		self.maxMemGB = userMaxMemGB
		util.LogInfo("jobmngr", "Using %d GB, per --localmem option.", self.maxMemGB)
		if cgMem > 0 && int64(userMaxMemGB)*1024*1024*1024 > cgMem {
			util.PrintInfo("jobmngr",
				"WARNING: User-supplied amount %d GB is higher than "+
					"the detected cgroup memory limit of %0.1f GB",
				userMaxMemGB, float64(cgMem)/(1024*1024*1024))
		}
	} else {
		MAXMEM_FRACTION := 0.9
		if cgMem > 0 && cgMem < sysMem.Total {
			util.LogInfo("jobmngr",
				"Detected cgroup memory limit of %d bytes.  Using it instead of total system memory %d",
				cgMem, sysMem.Total)
			sysMem.Total = cgMem
			if cgUse < cgMem && cgMem-cgUse < sysMem.ActualFree {
				sysMem.ActualFree = cgMem - cgUse
				MAXMEM_FRACTION = 0.96
			}
		}
		// Otherwise, set Max usable GB to MAXMEM_FRACTION * GB of total
		// memory reported by the system.
		if clusterMode {
			sysMemGB := int((sysMem.ActualFree + (1024*1024 - 1)) / (1024 * 1024 * 1024))
			if self.jobSettings.MemGBPerJob < sysMemGB {
				sysMemGB = self.jobSettings.MemGBPerJob
			}
			if sysMemGB < 1 {
				sysMemGB = 1
			}
			self.maxMemGB = sysMemGB
			if sysMemGB < self.jobSettings.MemGBPerJob {
				util.PrintInfo("jobmngr",
					"WARNING: Using %d GB for local jobs.  Recommended free memory is %d GB",
					self.maxMemGB, self.jobSettings.MemGBPerJob)
			} else {
				util.LogInfo("jobmngr", "Using %d GB for local jobs.", self.maxMemGB)
			}
		} else {
			sysMemGB := int(float64(sysMem.Total) * MAXMEM_FRACTION / 1073741824)
			// Set floor to 1GB.
			if sysMemGB < 1 {
				sysMemGB = 1
			}
			self.maxMemGB = sysMemGB
			util.LogInfo("jobmngr", "Using %d GB, %d%% of system memory.",
				self.maxMemGB,
				int(MAXMEM_FRACTION*100))
		}
	}

	if int64(self.maxMemGB*1024) > (sysMem.ActualFree+(1024*1024-1))/(1024*1024) {
		util.PrintInfo("jobmngr",
			"WARNING: configured to use %dGB of local memory, but only %.1fGB is currently available.",
			self.maxMemGB, float64(sysMem.ActualFree+(1024*1024-1))/(1024*1024*1024))
	}
	if cgSoftLimit != 0 && int64(self.maxMemGB)*1024*1024*1024 > cgSoftLimit {
		util.PrintInfo("jobmngr",
			"WARNING: detected a cgroup soft memory limit of %.1fGB. "+
				"If the system runs low on memory, jobs may get killed.",
			float64(cgSoftLimit)/(1024*1024*1024))
	}
	self.maxVmemMB = int64(CheckMaxVmem(
		uint64(1+self.maxMemGB)*uint64(self.highMem.Vmem+1024*1024*1024)) /
		(1024 * 1024))

	if self.maxVmemMB == 0 ||
		int64(userMaxVMemGB)*1024 < self.maxVmemMB {
		self.maxVmemMB = int64(userMaxVMemGB) * 1024
	}
	// Subtract off mrp's current vmem usage.  If mrp is running in e.g. and
	// SGE cluster job with h_vmem set, the process will be killed if vmem
	// for the whole tree exceeds the value given, so we need to make sure to
	// reserve some space for mrp itself.  But, make sure at least 1gb remains
	// or else it'll just hang on local jobs.
	//
	// Subtracting it here is not ideal, since mrp's vmem usage may increase
	// over time.  However as we update the process tree memory usage later on
	// we ignore mrp's usage because for rss that can cause the pipeline to
	// hang.
	if selfMem := self.highMem.Vmem / (1024 * 1024); selfMem+1024 < self.maxVmemMB {
		self.maxVmemMB -= selfMem
	}
	requiredVmemGB := int64(self.jobSettings.MemGBPerJob+self.jobSettings.ExtraVmemGB) +
		(self.highMem.Vmem+1024*1024*1024-1)/(1024*1024*1024)
	if self.maxVmemMB > 0 && self.maxVmemMB/1024 < requiredVmemGB {
		util.PrintInfo("jobmngr",
			"WARNING: mrp will not run correctly with less"+
				"\n                              "+
				"than %dGB of virtual address space available.",
			requiredVmemGB)
	}
}

func formatCentiThreads(size int64) string {
	if size%100 == 0 {
		return string(append(
			strconv.AppendInt(make([]byte, 0, 64), size/100, 10),
			" threads"...))
	}
	buf := make([]byte, 0, 18+len(" threads"))
	buf = strconv.AppendFloat(buf, float64(size)/100, 'g', 3, 64)
	return string(append(buf, " threads"...))
}

func formatMemMB(size int64) string {
	if size < 1024 {
		return string(append(
			strconv.AppendInt(make([]byte, 0, 64), size/100, 10),
			" MB of memory"...))
	}
	buf := make([]byte, 0, 18+len(" GB of memory"))
	buf = strconv.AppendFloat(buf, float64(size)/1024, 'g', 3, 64)
	return string(append(buf, " GB of memory"...))
}

func formatVMemMB(size int64) string {
	if size < 1024 {
		return string(append(
			strconv.AppendInt(make([]byte, 0, 64), size/100, 10),
			" MB of address space"...))
	}
	buf := make([]byte, 0, 18+len(" GB of address space"))
	buf = strconv.AppendFloat(buf, float64(size)/1024, 'g', 3, 64)
	return string(append(buf, " GB of address space"...))
}

func (self *LocalJobManager) setupSemaphores() {
	self.centcoreSem = NewResourceSemaphore(int64(self.maxCores)*100, formatCentiThreads)
	self.memMBSem = NewResourceSemaphore(int64(self.maxMemGB)*1024, formatMemMB)
	if self.maxVmemMB > 0 {
		self.vmemMBSem = NewResourceSemaphore(self.maxVmemMB, formatVMemMB)
	}
	if rlim, err := GetMaxProcs(); err != nil {
		util.LogError(err, "jobmngr",
			"WARNING: Could not get process rlimit.")
	} else if rlimMax(rlim) > startingThreadCount &&
		rlimCur(rlim) > startingThreadCount {
		self.procsSem = NewResourceSemaphore(rlimMax(rlim), DefaultResourceFormatter("processes"))
		if err := self.procsSem.Acquire(startingThreadCount); err != nil {
			util.LogError(err, "runtime", "WARNING: attempting to launch a "+
				"process which may use more threads than the current ulimit "+
				"permits.")
		}

		if userProcs, err := GetUserProcessCount(); err != nil {
			self.procsSem.UpdateSize(rlimCur(rlim))
		} else {
			self.procsSem.UpdateFreeUsed(
				rlimCur(rlim)-int64(userProcs),
				startingThreadCount)
		}
		if self.procsSem.Available()/(procsPerJob+1) < int64(self.maxCores) {
			if rlimMax(rlim) > rlimCur(rlim) {
				util.PrintInfo("jobmngr",
					"WARNING: The current process count limit %d is low. "+
						"To increase parallelism, set ulimit -u %d.",
					rlimCur(rlim), rlimMax(rlim))
			} else {
				util.PrintInfo("jobmngr",
					"WARNING: The current process count limit %d is low. "+
						"Contact your system administrator to increase it.",
					rlimCur(rlim))
			}
		}
	}
	self.queue = []*exec.Cmd{}
	util.RegisterSignalHandler(self)
	if usedMem, err := GetProcessTreeMemory(os.Getpid(), true, nil); err == nil {
		self.highMem.IncreaseTo(usedMem)
	}
}

func (self *LocalJobManager) GetSettings() *JobManagerSettings {
	return self.jobSettings
}

func (self *LocalJobManager) refreshResources(localMode bool) error {
	var sysMem MemInfo
	if err := sysMem.Get(); err != nil {
		return err
	}
	usedMem, err := GetProcessTreeMemory(os.Getpid(), false, nil)
	if err != nil {
		util.LogError(err, "jobmngr", "Error getting process tree memory usage.")
	}
	self.highMem.IncreaseTo(usedMem)
	memDiff := self.memMBSem.UpdateFreeUsed(
		(sysMem.ActualFree+1024*1024-1)/(1024*1024),
		(usedMem.Rss+1024*1024-1)/(1024*1024))
	if memDiff < -int64(self.maxMemGB)*1024/8 &&
		memDiff/128 < self.lastMemDiff &&
		(localMode || sysMem.ActualFree < 2*1024*1024*1024) {
		util.LogInfo("jobmngr", "%.1fGB less memory than expected was free", float64(-memDiff)/1024)
		if usedMem.Rss > self.memMBSem.Reserved()*1024*1024 {
			util.LogInfo("jobmngr",
				"MRP's child processes are using %.1fGB of rss, %.1f vmem.  %.1fGB are reserved.",
				float64(usedMem.Rss)/(1024*1024*1024),
				float64(usedMem.Vmem)/(1024*1024*1024),
				float64(self.memMBSem.Reserved())/1024)
		}
		cgLim, softLimit, cgUse := util.GetCgroupMemoryLimit()
		if cgLim > 0 && cgUse > self.memMBSem.Reserved()*1024*1024 {
			if softLimit > 0 && softLimit < cgLim {
				util.LogInfo("jobmngr",
					"cgroup rss usage is %.1fGB, vs. soft limit of %.1f",
					float64(cgUse)/(1024*1024*1024),
					float64(softLimit)/(1024*1024*1024))
			} else {
				util.LogInfo("jobmngr",
					"cgroup rss usage is %.1fGB, vs. limit of %.1f",
					float64(cgUse)/(1024*1024*1024),
					float64(cgLim)/(1024*1024*1024))
			}
		}
	}
	self.lastMemDiff = memDiff / 128
	if self.vmemMBSem != nil {
		self.vmemMBSem.UpdateActual(
			self.maxVmemMB - usedMem.Vmem/(1024*1024))
	}
	if self.limitLoad {
		var load LoadAverage
		if err := load.Get(); err != nil {
			return err
		}
		if diff := self.centcoreSem.UpdateActual(
			int64((float64(runtime.NumCPU()) - load.One + 0.9) * 100),
		); diff < -int64(self.maxCores)*100/4 &&
			localMode {
			util.LogInfo("jobmngr", "%g fewer core%s than expected were free.",
				-float64(diff)/100, util.Pluralize(int(-diff)))
		}
	}
	if self.procsSem != nil {
		if rlim, err := GetMaxProcs(); err != nil {
			return err
		} else if userProcs, err := GetUserProcessCount(); err != nil {
			return err
		} else {
			self.procsSem.UpdateFreeUsed(
				rlimCur(rlim)-int64(userProcs),
				int64(usedMem.Procs)+startingThreadCount)
		}
	}
	return nil
}

func (self *LocalJobManager) HandleSignal(sig os.Signal) {
	if self.highMem.Rss > 0 {
		if ser, err := json.MarshalIndent(self.highMem, "", "  "); err == nil {
			util.LogInfo("jobmngr", "Highest memory usage observed: %s", string(ser))
		}
	}
}

func (self *LocalJobManager) GetSystemReqs(request *JobResources) JobResources {
	result := *request
	// Sanity check and cap to self.maxCores.
	var centiCores int
	if result.Threads < 0 {
		centiCores = int(math.Floor(result.Threads * 100))
	} else {
		centiCores = int(math.Ceil(result.Threads * 100))
	}
	if centiCores == 0 {
		centiCores = self.jobSettings.ThreadsPerJob * 100
	} else if centiCores < 0 {
		centiCores = self.maxCores * 100
	}
	if centiCores > self.maxCores*100 {
		if self.debug {
			util.LogInfo("jobmngr", "Need %g core%s but settling for %d.",
				result.Threads,
				util.Pluralize(centiCores/100), self.maxCores)
		}
		result.Threads = float64(self.maxCores)
	} else {
		result.Threads = float64(centiCores) / 100
	}

	// Sanity check and cap to self.maxMemGB.
	var memMb, vmemMb int64
	if result.MemGB < 0 {
		memMb = int64(math.Floor(result.MemGB * 1024))
	} else {
		memMb = int64(math.Ceil(result.MemGB * 1024))
	}
	if memMb == 0 {
		memMb = int64(self.jobSettings.MemGBPerJob) * 1024
	} else {
		if memMb < 0 {
			avail := self.memMBSem.CurrentSize()
			if avail < 1 || avail < -memMb {
				memMb = -memMb
			} else {
				if self.debug {
					util.LogInfo("jobmngr",
						"Adaptive request for at least %d MB being given %d.",
						-memMb, avail)
				}
				memMb = avail
			}
		}
	}

	if result.VMemGB < 0 {
		vmemMb = int64(math.Floor(result.VMemGB * 1024))
	} else {
		vmemMb = int64(math.Ceil(result.VMemGB * 1024))
	}
	if vmemMb == 0 {
		vmemMb = memMb + int64(self.jobSettings.ExtraVmemGB)*1024
	}
	if vmemMb < 0 {
		if self.vmemMBSem != nil {
			avail := self.vmemMBSem.CurrentSize()
			if avail < 1 || avail < -vmemMb {
				vmemMb = -vmemMb
			} else {
				if self.debug {
					util.LogInfo("jobmngr",
						"Adaptive request for at least %d vmem MB being given %d.",
						-vmemMb, avail)
				}
				vmemMb = avail
			}
		}
	}

	// TODO: Stop allowing stages to ask for more than the max.  Require
	// stages which can adapt to the available memory to ask for a negative
	// amount as a sentinel.
	if memMb > int64(self.maxMemGB)*1024 {
		if self.debug {
			util.LogInfo("jobmngr",
				"Need %d MB but settling for %d GB.",
				memMb,
				self.maxMemGB)
		}
		util.LogInfo(
			"jobmngr",
			"Job asked for %d MB but is being given %d GB.\n"+
				"This behavior is deprecated - jobs which can adapt "+
				"their memory usage should ask for -%g.",
			memMb, self.maxMemGB, result.MemGB)
		memMb = int64(self.maxMemGB) * 1024
	}
	if self.maxVmemMB > 0 && vmemMb > self.maxVmemMB {
		if self.debug {
			util.LogInfo("jobmngr",
				"Need %d MB of vmem but settling for %d.",
				vmemMb,
				self.maxVmemMB)
		}
		util.LogInfo(
			"jobmngr",
			"Job asked for %d MB but is being given %d of vmem.\n"+
				"This behavior is deprecated - jobs which can adapt "+
				"their memory usage should ask for -%g.",
			vmemMb, self.maxVmemMB, result.VMemGB)
		vmemMb = self.maxVmemMB
	}
	if vmemMb > 0 && vmemMb < memMb {
		vmemMb = memMb
	}
	result.MemGB = float64(memMb) / 1024
	result.VMemGB = float64(vmemMb) / 1024

	return result
}

func (self *LocalJobManager) checkQueue(ids []string, _ context.Context) ([]string, string) {
	return ids, ""
}

func (self *LocalJobManager) hasQueueCheck() bool {
	return false
}

func (self *LocalJobManager) queueCheckGrace() time.Duration {
	return 0
}

func (self *LocalJobManager) Enqueue(shellCmd string, argv []string,
	envs map[string]string, metadata *Metadata, resRequest *JobResources,
	fqname string, retries int, waitTime int, localpreflight bool) {
	enc := func() {
		r := trace.StartRegion(context.Background(), "queueLocal")
		defer r.End()

		res := self.GetSystemReqs(resRequest)

		// Exec the shell directly.
		cmd := exec.Command(shellCmd, argv...)
		cmd.Dir = metadata.curFilesPath
		if self.maxCores < runtime.NumCPU() {
			// If, and only if, the user specified a core limit less than the
			// detected core count, make sure jobs actually don't use more
			// threads than they're supposed to.
			cmd.Env = util.MergeEnv(threadEnvs(self,
				int(math.Ceil(res.Threads)), envs))
		} else {
			// In this case it's ok if we oversubscribe a bit since we're
			// (probably) not sharing the machine.
			cmd.Env = util.MergeEnv(envs)
		}

		stdoutPath := metadata.MetadataFilePath("stdout")
		stderrPath := metadata.MetadataFilePath("stderr")

		// Acquire cores.
		if self.debug {
			util.LogInfo("jobmngr",
				"Waiting for %g core%s",
				res.Threads,
				util.PluralizeFloat(res.Threads))
		}
		centiCores := int64(math.Ceil(res.Threads * 100))
		if err := self.centcoreSem.Acquire(centiCores); err != nil {
			util.LogError(err, "jobmngr",
				"%s requested %g threads, but the job manager was only configured to use %d.",
				metadata.fqname, res.Threads, self.maxCores)
			metadata.WriteErrorString(err.Error())
			return
		}
		defer func(centiCores int64) {
			// Release cores.
			self.centcoreSem.Release(centiCores)
			if self.debug {
				threads := float64(centiCores) / 100
				util.LogInfo("jobmngr", "Released %g core%s (%g/%d in use)",
					threads,
					util.PluralizeFloat(threads),
					float64(self.centcoreSem.InUse())/100,
					self.maxCores)
			}
		}(centiCores)

		if self.debug {
			threads := float64(centiCores) / 100
			util.LogInfo("jobmngr",
				"Acquired %g core%s (%g/%d in use)",
				threads,
				util.PluralizeFloat(threads),
				float64(self.centcoreSem.InUse())/100,
				self.maxCores)
		}

		// Acquire memory.
		if self.debug {
			util.LogInfo("jobmngr",
				"Waiting for %g GB",
				res.MemGB)
		}
		memMb := int64(math.Ceil(res.MemGB * 1024))
		if err := self.memMBSem.Acquire(memMb); err != nil {
			util.LogError(err, "jobmngr",
				"%s requested %g GB of memory, but the job manager was only configured to use %d.",
				metadata.fqname, res.MemGB, self.maxMemGB)
			metadata.WriteErrorString(err.Error())
			return
		}
		defer func(memMb int64) {
			// Release memory.
			self.memMBSem.Release(memMb)
			if self.debug {
				util.LogInfo("jobmngr", "Released %g GB (%.1f/%d in use)",
					float64(memMb)/1024,
					float64(self.memMBSem.InUse())/1024, self.maxMemGB)
			}
		}(memMb)

		if self.debug {
			util.LogInfo("jobmngr",
				"Acquired %g GB (%.1f/%d in use)",
				res.MemGB,
				float64(self.memMBSem.InUse())/1024, self.maxMemGB)
		}

		if sem := self.vmemMBSem; sem != nil {
			// Acquire vmem
			vmem := int64(res.VMemGB) * 1024
			if err := sem.Acquire(vmem); err != nil {
				util.LogError(err, "jobmngr",
					"%s requested %d GB of virtual memory, but the "+
						"job manager was only configured to use %.1f.",
					metadata.fqname, res.VMemGB, float64(self.maxVmemMB)/1024)
				metadata.WriteErrorString(err.Error())
				return
			}
			defer func(vmem int64, sem *ResourceSemaphore) {
				// Release memory.
				sem.Release(vmem)
				if self.debug {
					util.LogInfo("jobmngr", "Released %.1f GB (%.1f/%.1f in use)",
						float64(vmem)/1024,
						float64(sem.InUse())/1024,
						float64(self.maxVmemMB)/1024)
				}
			}(vmem, sem)
			if self.debug {
				util.LogInfo("jobmngr",
					"Acquired %.1f virtual GB (%.1f/%.1f in use)",
					res.VMemGB,
					float64(sem.InUse())/1024,
					float64(self.maxVmemMB)/1024)
			}
		}

		if self.debug {
			util.LogInfo("jobmngr", "%d goroutines", runtime.NumGoroutine())
		}

		if self.procsSem != nil {
			procEstimate := procsPerJob + (centiCores+99)/100
			// Acquire processes
			if self.debug {
				util.LogInfo("jobmngr", "Waiting for %d processes", procEstimate)
			}
			if err := self.procsSem.Acquire(procEstimate); err != nil {
				util.LogError(err, "jobmngr",
					"%s estimated to require %d processes, but the process ulimit is %d.",
					metadata.fqname, procEstimate, self.procsSem.CurrentSize())
				metadata.WriteErrorString(err.Error())
				return
			}

			defer func(procEstimate int64) {
				// Release processes.
				self.procsSem.Release(procEstimate)
				if self.debug {
					util.LogInfo("jobmngr", "Released %d processes (%d/%d in use)",
						procEstimate, self.procsSem.InUse(), self.procsSem.CurrentSize())
				}
			}(procEstimate)
			if self.debug {
				util.LogInfo("jobmngr", "Acquired %d processes (%d/%d in use)",
					procEstimate, self.procsSem.InUse(), self.procsSem.CurrentSize())
			}
			if self.debug {
				util.LogInfo("jobmngr", "%d goroutines", runtime.NumGoroutine())
			}
		}
		err := executeLocal(cmd, stdoutPath, stderrPath, localpreflight, metadata)
		// CentOS < 5.5 workaround
		if err != nil {
			if strings.Contains(err.Error(), exitCodeString) {
				retries += 1
				if waitTime == 0 {
					waitTime = 2
				} else {
					waitTime *= 2
				}
			} else {
				retries = maxRetries + 1
			}
			if retries > maxRetries {
				if _, err2 := metadata.readRawSafe(Errors); os.IsNotExist(err2) {
					// Only write _errors if the job didn't write one before
					// failing.  Because this is local mode, we don't need to
					// worry about nfs data races.
					metadata.WriteErrorString(err.Error())
				}
			} else {
				util.LogInfo("jobmngr",
					"Job failed: %s. Retrying job %s in %d seconds",
					err.Error(), fqname, waitTime)
				self.Enqueue(shellCmd, argv, envs, metadata, resRequest,
					fqname, retries,
					waitTime, localpreflight)
			}
		} else {
			// Notify
			select {
			case self.jobDone <- struct{}{}:
			default:
			}
		}
	}
	if waitTime > 0 {
		time.AfterFunc(time.Second*time.Duration(waitTime), enc)
	} else {
		go enc()
	}
}

func executeLocal(cmd *exec.Cmd, stdoutPath, stderrPath string,
	localpreflight bool, metadata *Metadata) error {
	if err := func(cmd *exec.Cmd, stdoutPath, stderrPath string,
		localpreflight bool, metadata *Metadata) error {
		// Set up _stdout and _stderr for the job.
		stdoutFile, err := os.Create(stdoutPath)
		if err == nil {
			if _, err := stdoutFile.WriteString("[stdout]\n"); err != nil {
				util.LogError(err, "jobmngr", "Error writing job stdout header")
			}
			// If local preflight stage, let stdout go to the console
			if localpreflight {
				cmd.Stdout = os.Stdout
			} else {
				cmd.Stdout = stdoutFile
			}
			defer stdoutFile.Close()
		} else {
			util.LogError(err, "jobmngr", "Error creating job stdout file.")
		}
		cmd.SysProcAttr = util.Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGTERM)
		stderrFile, err := os.Create(stderrPath)
		if err == nil {
			if _, err := stderrFile.WriteString("[stderr]\n"); err != nil {
				util.LogError(err, "jobmngr", "Error writing job stderr header")
			}
			cmd.Stderr = stderrFile
			defer stderrFile.Close()
		} else {
			util.LogError(err, "jobmngr", "Error creating job stderr file.")
		}

		// Run the command and wait for completion.
		util.EnterCriticalSection()
		defer util.ExitCriticalSection()
		err = cmd.Start()
		if err == nil {
			return metadata.remove(QueuedLocally)
		}
		return err
	}(cmd, stdoutPath, stderrPath,
		localpreflight, metadata); err != nil {
		return err
	}
	return cmd.Wait()
}

// Done returns a channel which gets notified when a local job exits.
func (self *LocalJobManager) Done() <-chan struct{} {
	return self.jobDone
}

func (self *LocalJobManager) GetMaxCores() int {
	return self.maxCores
}

func (self *LocalJobManager) GetMaxMemGB() int {
	return self.maxMemGB
}

func (self *LocalJobManager) GetMaxVMemGB() int {
	return int(self.maxVmemMB / 1024)
}

func (self *LocalJobManager) execJob(shellCmd string, argv []string,
	envs map[string]string, metadata *Metadata, resRequest *JobResources,
	fqname string, shellName string, preflight bool) {
	self.Enqueue(shellCmd, argv, envs, metadata, resRequest, fqname, 0, 0, preflight)
}

func (self *LocalJobManager) endJob(*Metadata) {}
