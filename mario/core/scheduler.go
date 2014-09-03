//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Margo scheduler for local cores.
//
package core

import (
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
	cores int
	sem   semaphore
	queue []*exec.Cmd
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func NewScheduler(reqCores int) *Scheduler {
	self := &Scheduler{}
	self.cores = runtime.NumCPU()
	if reqCores < self.cores {
		self.cores = reqCores
	}
	LogInfo("schedlr", "%d of %d available core%s requested, using %d.",
		reqCores, runtime.NumCPU(), pluralize(reqCores), self.cores)
	self.sem = make(semaphore, self.cores)
	self.queue = []*exec.Cmd{}
	return self
}

func (self *Scheduler) Enqueue(cmd *exec.Cmd, threads int, stdoutFile *os.File, stderrFile *os.File) {
	go func(cmd *exec.Cmd, threads int, stdoutFile *os.File, stderrFile *os.File) {
		defer stdoutFile.Close()
		defer stderrFile.Close()
		if threads > self.cores {
			LogInfo("schedlr", "Need %d core%s but settling for %d.", threads, pluralize(threads), self.cores)
			threads = self.cores
		}
		LogInfo("schedlr", "Waiting for %d core%s.", threads, pluralize(threads))
		self.sem.P(threads)
		LogInfo("schedlr", "Acquiring %d core%s.", threads, pluralize(threads))
		cmd.Start()
		cmd.Wait()
		LogInfo("schedlr", "Releasing %d core%s.", threads, pluralize(threads))
		self.sem.V(threads)
	}(cmd, threads, stdoutFile, stderrFile)
}
