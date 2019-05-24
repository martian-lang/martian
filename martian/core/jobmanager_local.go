// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"runtime/trace"
	"strings"
	"syscall"
	"time"

	sigar "github.com/cloudfoundry/gosigar"
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
	coreSem     *ResourceSemaphore
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
	sysMem := sigar.Mem{}
	sysMem.Get()
	cgMem, cgSoftLimit, cgUse := util.GetCgroupMemoryLimit()

	// Set Max GB of memory usable at one time.
	if userMaxMemGB > 0 {
		// If user specified --localmem, use that value for Max usable GB.
		self.maxMemGB = userMaxMemGB
		util.LogInfo("jobmngr", "Using %d GB, per --localmem option.", self.maxMemGB)
		if cgMem > 0 && int64(userMaxMemGB)*1024*1024*1024 > cgMem {
			util.PrintInfo("jobmngr",
				"WARNING: User-supplied amount %d GB is higher than the detected cgroup memory limit of %0.1f GB",
				userMaxMemGB, float64(cgMem)/(1024*1024*1024))
		}
	} else {
		MAXMEM_FRACTION := 0.9
		if cgMem > 0 && cgMem < int64(sysMem.Total) {
			util.LogInfo("jobmngr",
				"Detected cgroup memory limit of %d bytes.  Using it instead of total system memory %d",
				cgMem, sysMem.Total)
			sysMem.Total = uint64(cgMem)
			if cgUse < cgMem && cgMem-cgUse < int64(sysMem.ActualFree) {
				sysMem.ActualFree = uint64(cgMem - cgUse)
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
			util.LogInfo("jobmngr", "Using %d GB, %d%% of system memory.", self.maxMemGB,
				int(MAXMEM_FRACTION*100))
		}
	}

	if uint64(self.maxMemGB*1024) > (sysMem.ActualFree+(1024*1024-1))/(1024*1024) {
		util.PrintInfo("jobmngr",
			"WARNING: configured to use %dGB of local memory, but only %.1fGB is currently available.",
			self.maxMemGB, float64(sysMem.ActualFree+(1024*1024-1))/(1024*1024*1024))
	}
	if cgSoftLimit != 0 && int64(self.maxMemGB)*1024*1024*1024 > cgSoftLimit {
		util.PrintInfo("jobmngr",
			"WARNING: detected a cgroup soft memory limit of %.1fGB. If the system runs low on memory, jobs may get killed.",
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

func (self *LocalJobManager) setupSemaphores() {
	self.coreSem = NewResourceSemaphore(int64(self.maxCores), "threads")
	self.memMBSem = NewResourceSemaphore(int64(self.maxMemGB)*1024, "MB of memory")
	if self.maxVmemMB > 0 {
		self.vmemMBSem = NewResourceSemaphore(self.maxVmemMB,
			"MB of address space")
	}
	if rlim, err := GetMaxProcs(); err != nil {
		util.LogError(err, "jobmngr",
			"WARNING: Could not get process rlimit.")
	} else if int64(rlim.Max) > startingThreadCount &&
		int64(rlim.Cur) > startingThreadCount {
		self.procsSem = NewResourceSemaphore(int64(rlim.Max), "processes")
		self.procsSem.Acquire(startingThreadCount)

		if userProcs, err := GetUserProcessCount(); err != nil {
			self.procsSem.UpdateSize(int64(rlim.Cur))
		} else {
			self.procsSem.UpdateFreeUsed(
				int64(rlim.Cur)-int64(userProcs),
				startingThreadCount)
		}
		if self.procsSem.Available()/(procsPerJob+1) < int64(self.maxCores) {
			if rlim.Max > rlim.Cur {
				util.PrintInfo("jobmngr",
					"WARNING: The current process count limit %d is low. "+
						"To increase parallelism, set ulimit -u %d.",
					rlim.Cur, rlim.Max)
			} else {
				util.PrintInfo("jobmngr",
					"WARNING: The current process count limit %d is low. "+
						"Contact your system administrator to increase it.",
					rlim.Cur)
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
	sysMem := sigar.Mem{}
	if err := sysMem.Get(); err != nil {
		return err
	}
	usedMem, err := GetProcessTreeMemory(os.Getpid(), false, nil)
	if err != nil {
		util.LogError(err, "jobmngr", "Error getting process tree memory usage.")
	}
	self.highMem.IncreaseTo(usedMem)
	memDiff := self.memMBSem.UpdateFreeUsed(
		(int64(sysMem.ActualFree)+(1024*1024-1))/(1024*1024),
		(usedMem.Rss+(1024*1024-1))/(1024*1024))
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
		load := sigar.LoadAverage{}
		if err := load.Get(); err != nil {
			return err
		}
		if diff := self.coreSem.UpdateActual(int64(
			float64(runtime.NumCPU()) - load.One + 0.9)); diff < -int64(self.maxCores)/4 &&
			localMode {
			util.LogInfo("jobmngr", "%d fewer core%s than expected were free.", -diff, util.Pluralize(int(-diff)))
		}
	}
	if self.procsSem != nil {
		if rlim, err := GetMaxProcs(); err != nil {
			return err
		} else if userProcs, err := GetUserProcessCount(); err != nil {
			return err
		} else {
			self.procsSem.UpdateFreeUsed(
				int64(rlim.Cur)-int64(userProcs),
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
	if result.Threads == 0 {
		result.Threads = self.jobSettings.ThreadsPerJob
	} else if result.Threads < 0 {
		result.Threads = self.maxCores
	}
	if result.Threads > self.maxCores {
		if self.debug {
			util.LogInfo("jobmngr", "Need %d core%s but settling for %d.",
				result.Threads,
				util.Pluralize(result.Threads), self.maxCores)
		}
		result.Threads = self.maxCores
	}

	// Sanity check and cap to self.maxMemGB.
	if result.MemGB == 0 {
		result.MemGB = self.jobSettings.MemGBPerJob
	}
	if result.MemGB < 0 {
		avail := int(self.memMBSem.CurrentSize() / 1024)
		if avail < 1 || avail < -result.MemGB {
			result.MemGB = -result.MemGB
		} else {
			if self.debug {
				util.LogInfo("jobmngr",
					"Adaptive request for at least %d GB being given %d.",
					-result.MemGB, avail)
			}
			result.MemGB = avail
		}
	}

	if result.VMemGB == 0 {
		result.VMemGB = result.MemGB + self.jobSettings.ExtraVmemGB
	}
	if result.VMemGB < 0 {
		if self.vmemMBSem != nil {
			avail := int(self.vmemMBSem.CurrentSize() / 1024)
			if avail < 1 || avail < -result.VMemGB {
				result.VMemGB = -result.VMemGB
			} else {
				if self.debug {
					util.LogInfo("jobmngr",
						"Adaptive request for at least %d vmem GB being given %d.",
						-result.VMemGB, avail)
				}
				result.VMemGB = avail
			}
		}
	}

	// TODO: Stop allowing stages to ask for more than the max.  Require
	// stages which can adapt to the available memory to ask for a negative
	// amount as a sentinel.
	if result.MemGB > self.maxMemGB {
		if self.debug {
			util.LogInfo("jobmngr",
				"Need %d GB but settling for %d.",
				result.MemGB,
				self.maxMemGB)
		}
		util.LogInfo(
			"jobmngr",
			"Job asked for %d GB but is being given %d.\n"+
				"This behavior is deprecated - jobs which can adapt "+
				"their memory usage should ask for -%d.",
			result.MemGB, self.maxMemGB, result.MemGB)
		result.MemGB = self.maxMemGB
	}
	if self.maxVmemMB > 0 && int64(result.VMemGB)*1024 > self.maxVmemMB {
		if self.debug {
			util.LogInfo("jobmngr",
				"Need %d GB of vmem but settling for %d.",
				result.VMemGB,
				self.maxVmemMB/1024)
		}
		util.LogInfo(
			"jobmngr",
			"Job asked for %d GB but is being given %d of vmem.\n"+
				"This behavior is deprecated - jobs which can adapt "+
				"their memory usage should ask for -%d.",
			result.VMemGB, self.maxVmemMB/1024, result.VMemGB)
		result.VMemGB = int(self.maxVmemMB / 1024)
	}
	if result.VMemGB > 0 && result.VMemGB < result.MemGB {
		result.VMemGB = result.MemGB
	}

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
			cmd.Env = util.MergeEnv(threadEnvs(self, res.Threads, envs))
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
				"Waiting for %d core%s",
				res.Threads,
				util.Pluralize(res.Threads))
		}
		if err := self.coreSem.Acquire(int64(res.Threads)); err != nil {
			util.LogError(err, "jobmngr",
				"%s requested %d threads, but the job manager was only configured to use %d.",
				metadata.fqname, res.Threads, self.maxCores)
			metadata.WriteRaw(Errors, err.Error())
			return
		}
		defer func(threads int) {
			// Release cores.
			self.coreSem.Release(int64(threads))
			if self.debug {
				util.LogInfo("jobmngr", "Released %d core%s (%d/%d in use)", threads,
					util.Pluralize(threads), self.coreSem.InUse(), self.maxCores)
			}
		}(res.Threads)

		if self.debug {
			util.LogInfo("jobmngr",
				"Acquired %d core%s (%d/%d in use)",
				res.Threads,
				util.Pluralize(res.Threads),
				self.coreSem.InUse(),
				self.maxCores)
		}

		// Acquire memory.
		if self.debug {
			util.LogInfo("jobmngr",
				"Waiting for %d GB",
				res.MemGB)
		}
		if err := self.memMBSem.Acquire(int64(res.MemGB) * 1024); err != nil {
			util.LogError(err, "jobmngr",
				"%s requested %d GB of memory, but the job manager was only configured to use %d.",
				metadata.fqname, res.MemGB, self.maxMemGB)
			metadata.WriteRaw(Errors, err.Error())
			return
		}
		defer func(memGB int) {
			// Release memory.
			self.memMBSem.Release(int64(memGB) * 1024)
			if self.debug {
				util.LogInfo("jobmngr", "Released %d GB (%.1f/%d in use)", memGB,
					float64(self.memMBSem.InUse())/1024, self.maxMemGB)
			}
		}(res.MemGB)

		if self.debug {
			util.LogInfo("jobmngr",
				"Acquired %d GB (%.1f/%d in use)",
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
				metadata.WriteRaw(Errors, err.Error())
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
					"Acquired %d virtual GB (%.1f/%.1f in use)",
					res.VMemGB,
					float64(sem.InUse())/1024,
					float64(self.maxVmemMB)/1024)
			}
		}

		if self.debug {
			util.LogInfo("jobmngr", "%d goroutines", runtime.NumGoroutine())
		}

		if self.procsSem != nil {
			procEstimate := int64(procsPerJob + res.Threads)
			// Acquire processes
			if self.debug {
				util.LogInfo("jobmngr", "Waiting for %d processes", procEstimate)
			}
			if err := self.procsSem.Acquire(procEstimate); err != nil {
				util.LogError(err, "jobmngr",
					"%s estimated to require %d processes, but the process ulimit is %d.",
					metadata.fqname, procEstimate, self.procsSem.CurrentSize())
				metadata.WriteRaw(Errors, err.Error())
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
					metadata.WriteRaw(Errors, err.Error())
				}
			} else {
				util.LogInfo("jobmngr",
					"Job failed: %s. Retrying job %s in %d seconds",
					err.Error(), fqname, waitTime)
				self.Enqueue(shellCmd, argv, envs, metadata, resRequest, fqname, retries,
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
	// Set up _stdout and _stderr for the job.
	if stdoutFile, err := os.Create(stdoutPath); err == nil {
		stdoutFile.WriteString("[stdout]\n")
		// If local preflight stage, let stdout go to the console
		if localpreflight {
			cmd.Stdout = os.Stdout
		} else {
			cmd.Stdout = stdoutFile
		}
		defer stdoutFile.Close()
	}
	cmd.SysProcAttr = util.Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGTERM)
	if stderrFile, err := os.Create(stderrPath); err == nil {
		stderrFile.WriteString("[stderr]\n")
		cmd.Stderr = stderrFile
		defer stderrFile.Close()
	}

	// Run the command and wait for completion.
	if err := func(metadata *Metadata, cmd *exec.Cmd) error {
		util.EnterCriticalSection()
		defer util.ExitCriticalSection()
		err := cmd.Start()
		if err == nil {
			metadata.remove(QueuedLocally)
		}
		return err
	}(metadata, cmd); err != nil {
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
