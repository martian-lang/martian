//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo scheduler for local cores.
//
package core

import (
	"github.com/cloudfoundry/gosigar"
	"os"
	"os/exec"
	"runtime"
)

type semaphore chan bool

func (s semaphore) P(n int) {
	for i := 0; i < n; i++ {
		s <- true
	}
}

func (s semaphore) V(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}

type Scheduler struct {
	cores    int
	memGB    int
	coreSem  semaphore
	memGBSem semaphore
	queue    []*exec.Cmd
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func NewScheduler(reqCores int, reqMem int) *Scheduler {
	MAXMEM_FRACTION := 0.75

	self := &Scheduler{}

	self.cores = runtime.NumCPU()
	if reqCores > 0 {
		self.cores = reqCores
		LogInfo("schedlr", "Using %d core%s, per --maxcores option.",
			self.cores, pluralize(self.cores))
	} else {
		LogInfo("schedlr", "Using %d core%s available on system.",
			self.cores, pluralize(self.cores))
	}

	mem := sigar.Mem{}
	mem.Get()
	self.memGB = int(float64(mem.Total) * MAXMEM_FRACTION / 1000000000)
	if reqMem > 0 {
		self.memGB = reqMem
		LogInfo("schedlr", "Using %d GB, per --maxmem option.", self.memGB)
	} else {
		LogInfo("schedlr", "Using %d GB, %d%% of system memory.", self.memGB,
			int(MAXMEM_FRACTION*100))
	}

	self.coreSem = make(semaphore, self.cores)
	self.memGBSem = make(semaphore, self.memGB)
	self.queue = []*exec.Cmd{}
	return self
}

func (self *Scheduler) Enqueue(cmd *exec.Cmd, threads int, memGB int,
	stdoutFile *os.File, stderrFile *os.File) {

	go func() {
		defer stdoutFile.Close()
		defer stderrFile.Close()
		if threads > self.cores {
			LogInfo("schedlr", "Need %d core%s but settling for %d.", threads,
				pluralize(threads), self.cores)
			threads = self.cores
		}
		if memGB > self.memGB {
			LogInfo("schedlr", "Need %d GB but settling for %d.", memGB,
				self.memGB)
			memGB = self.memGB
		}

		// Acquire cores.
		LogInfo("schedlr", "Waiting for %d core%s.", threads, pluralize(threads))
		self.coreSem.P(threads)
		LogInfo("schedlr", "Acquiring %d core%s (%d in use).", threads,
			pluralize(threads), len(self.coreSem))

		// Acquire memory.
		LogInfo("schedlr", "Waiting for %d GB.", memGB)
		self.memGBSem.P(memGB)
		LogInfo("schedlr", "Acquiring %d GB (%d in use).", memGB,
			len(self.memGBSem))

		cmd.Start()
		cmd.Wait()

		self.coreSem.V(threads)
		LogInfo("schedlr", "Releasing %d core%s (%d in use).", threads,
			pluralize(threads), len(self.coreSem))

		self.memGBSem.V(memGB)
		LogInfo("schedlr", "Releasing %d GB (%d in use).", memGB,
			len(self.memGBSem))
	}()
}

func (self *Scheduler) getCores() int {
	return self.cores
}

func (self *Scheduler) getMemGB() int {
	return self.memGB
}
