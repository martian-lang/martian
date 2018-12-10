//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//

// Martian job monitor.
//
// Manages process lifetime and data collection for martian stage code.
//
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/shlex"
	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/util"
)

const HeartbeatInterval = time.Minute * 2
const MemorySampleInterval = time.Second * 5

type runner struct {
	job         *exec.Cmd
	log         *os.File
	errorReader *os.File
	highMem     core.ObservedMemory
	ioStats     *core.IoStatsBuilder
	metadata    *core.Metadata
	runType     string
	jobInfo     *core.JobInfo
	start       time.Time
	isDone      chan struct{}
	perfDone    <-chan struct{}
}

func main() {
	util.SetupSignalHandlers()
	if len(os.Args) < 6 {
		panic("Insufficient arguments.\n" +
			"Expected: mrjob <exe> [exe args...] <split|main|join> " +
			"<metadata_path> <files_path> <journal_prefix>")
	}
	args := os.Args[len(os.Args)-4:]
	runType := args[0]
	metadataPath := args[1]
	filesPath := args[2]
	fqname := path.Base(args[3])
	journalPath := path.Dir(args[3])

	run := runner{
		ioStats:  core.NewIoStatsBuilder(),
		metadata: core.NewMetadataRunWithJournalPath(fqname, metadataPath, filesPath, journalPath, runType),
		runType:  runType,
		start:    time.Now(),
	}
	util.RegisterSignalHandler(&run)
	if log, err := os.OpenFile(run.metadata.MetadataFilePath(core.LogFile),
		os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
		run.Fail(err, "Can't open log file.")
	} else {
		run.log = log
		util.LogTeeWriter(log)
		defer run.log.Close()
	}
	run.metadata.UpdateJournal(core.StdOut)
	run.metadata.UpdateJournal(core.StdErr)

	run.Init()
	if err := run.StartJob(os.Args[1:]); err != nil {
		run.Fail(err, "Error starting job.")
	}
	run.isDone = make(chan struct{})
	run.WaitLoop()
}

func (self *runner) Init() {
	// In case the job template was wrong, set the working directory now.
	if err := os.Chdir(self.metadata.FilesPath()); err != nil {
		self.Fail(err, "Could not change to the correct working directory")
	}
	self.writeJobinfo()
	util.LogInfo("time", "__start__")
	if jErr := self.metadata.UpdateJournal(core.LogFile); jErr != nil {
		util.PrintError(jErr, "monitor",
			"Could not update log journal file.  Continuing, hoping for the best.")
	}
	// Check that the vmem limit is enough for the parent process plus
	// the a half gigabyte of margin over the job's physical memory
	// requirement.
	mem, _ := core.GetProcessTreeMemory(self.jobInfo.Pid, true, nil)
	core.CheckMaxVmem(
		uint64(self.jobInfo.MemGB*1024+512)*1024*1024 + uint64(mem.Vmem))
	self.setRlimit()
}

func getClusterEnv() map[string]string {
	re := regexp.MustCompile("^(?:EGO|SGE|LS[BF]|PBS|SLURM|JOB)_[^O]")
	captures := make(map[string]string)
	for _, env := range os.Environ() {
		sep := strings.Index(env, "=")
		if sep > 0 && re.MatchString(env[:sep]) {
			captures[env[:sep]] = env[sep+1:]
		}
	}
	if len(captures) == 0 {
		return nil
	} else {
		return captures
	}
}

func (self *runner) writeJobinfo() {
	jobInfo := &core.JobInfo{}
	if err := self.metadata.ReadInto(core.JobInfoFile, jobInfo); err != nil {
		self.Fail(err, "Error reading jobInfo.")
	} else {
		self.jobInfo = jobInfo
	}
	self.jobInfo.Cwd = self.metadata.FilesPath()
	self.jobInfo.Host, _ = os.Hostname()
	self.jobInfo.Pid = os.Getpid()
	self.jobInfo.ClusterEnv = getClusterEnv()
	if err := self.metadata.WriteAtomic(core.JobInfoFile, self.jobInfo); err != nil {
		self.Fail(err, "Could not write updated jobInfo.")
	}
}

func (self *runner) setRlimit() {
	if err := core.MaximizeMaxFiles(); err != nil {
		util.PrintError(err, "monitor", "Error setting the file rlimit.")
		return
	}
}

func (self *runner) done() {
	util.LogInfo("time", "__end__")
	// refresh jobInfo if possible, but if we can't that's ok.
	self.metadata.ReadInto(core.JobInfoFile, self.jobInfo)
	if self.jobInfo != nil {
		end := time.Now()
		self.jobInfo.WallClockInfo = &core.WallClockInfo{
			Start:    self.start.Format(util.TIMEFMT),
			End:      end.Format(util.TIMEFMT),
			Duration: end.Sub(self.start).Seconds(),
		}
		self.jobInfo.RusageInfo = core.GetRusage()
		if !self.highMem.IsZero() {
			self.jobInfo.MemoryUsage = &self.highMem
		}
		self.jobInfo.IoStats = &self.ioStats.IoStats
		if err := self.metadata.WriteAtomic(core.JobInfoFile, self.jobInfo); err != nil {
			util.PrintError(err, "monitor", "Could not write final jobInfo.")
		} else {
			self.metadata.UpdateJournal(core.JobInfoFile)
		}
	}
}

func (self *runner) Fail(err error, message string) {
	self.done()
	errStr := err.Error()
	target := core.Errors
	if _, ok := err.(*stageReturnedError); !ok {
		errStr = fmt.Sprintf("%s\n\n%s\n", message, err.Error())
		fmt.Fprint(os.Stderr, errStr)
	} else {
		if strings.HasPrefix(errStr, "ASSERT:") {
			errStr = errStr[len("ASSERT:"):]
			target = core.Assert
		}
	}
	if writeError := self.metadata.WriteRaw(target, errStr); writeError != nil {
		util.PrintError(writeError, "monitor", "Could not write errors file.")
	}
	if jErr := self.metadata.UpdateJournal(target); jErr != nil {
		util.PrintError(jErr, "monitor", "Could not update %v journal file.", target)
	}
	self.waitForPerf()
	os.Exit(0)
}

// Wait for up to 15 seconds after the stage code terminates for perf record to
// terminate (if applicable).  Otherwise some cluster managers might kill perf
// as soon as the head process for the job (mrjob, in this case) terminates.
func (self *runner) waitForPerf() {
	if c := self.perfDone; c != nil {
		select {
		case <-c:
		case <-time.After(15 * time.Second):
		}
	}
}

func totalCpu(ru *core.RusageInfo) float64 {
	if ru == nil {
		return 0
	}
	var total float64
	if ru.Self != nil {
		total += ru.Self.UserTime + ru.Self.SystemTime
	}
	if ru.Children != nil {
		total += ru.Children.UserTime + ru.Children.SystemTime
	}
	return total
}

func (self *runner) Complete() {
	self.done()
	target := core.CompleteFile
	if self.jobInfo.Monitor == "monitor" {
		if t := time.Since(self.start); t > time.Minute*15 {
			if threads := totalCpu(self.jobInfo.RusageInfo) /
				t.Seconds(); threads > 1.5*float64(self.jobInfo.Threads) {
				target = core.Errors
				if writeError := self.metadata.WriteRaw(target, fmt.Sprintf(
					"Stage exceeded its threads quota (using %.1f, allowed %d)",
					threads,
					self.jobInfo.Threads)); writeError != nil {
					util.PrintError(writeError, "monitor", "Could not write errors file.")
				}
			}
		}
	}
	if target == core.CompleteFile {
		if writeError := self.metadata.WriteTime(core.CompleteFile); writeError != nil {
			util.PrintError(writeError, "monitor", "Could not write complete file.")
		}
	}
	self.sync()
	if jErr := self.metadata.UpdateJournal(target); jErr != nil {
		util.PrintError(jErr, "monitor", "Could not update %v journal file.", target)
	}
	self.waitForPerf()
	os.Exit(0)
}

func (self *runner) sync() {
	if self.runType == "split" {
		syncFile(self.metadata.MetadataFilePath(core.StageDefsFile))
	} else {
		syncFile(self.metadata.MetadataFilePath(core.OutsFile))
	}
	syncFile(path.Dir(self.metadata.FilePath("nil")))
	syncFile(path.Dir(self.metadata.MetadataFilePath(core.CompleteFile)))
}

func (self *runner) makeErrorPipe() (*os.File, error) {
	var err error
	var writer *os.File
	self.errorReader, writer, err = os.Pipe()
	return writer, err
}

func (self *runner) StartJob(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	if writer, err := self.makeErrorPipe(); err != nil {
		return err
	} else {
		cmd.ExtraFiles = []*os.File{self.log, writer}
		defer writer.Close()
	}
	// We really don't want the child outliving the parent.
	cmd.SysProcAttr = util.Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGKILL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if pc := self.jobInfo.ProfileConfig; pc != nil && len(pc.Env) > 0 {
		cmd.Env = pc.MakeEnv(
			self.metadata.MetadataFilePath(core.PerfData),
			self.metadata.MetadataFilePath(core.ProfileOut))
	}
	if err := func() error {
		util.EnterCriticalSection()
		defer util.ExitCriticalSection()
		self.job = cmd
		return self.job.Start()
	}(); err != nil {
		self.errorReader.Close()
		return err
	}
	if err := self.startProfile(); err != nil {
		util.PrintError(err, "monitor", "Could not start profiling.")
	}
	return nil
}

func (self *runner) startProfile() error {
	var cmd *exec.Cmd
	var journaledFiles []core.MetadataFileName
	if perfArgs := os.Getenv("MRO_PERF_ARGS"); perfArgs != "" &&
		self.jobInfo.ProfileMode == core.PerfRecordProfile {
		// For backwards compatibility, ignore the custom config.
		journaledFiles = []core.MetadataFileName{core.PerfData}
		if args, err := shlex.Split(perfArgs); err != nil {
			util.PrintError(err, "profile", "Error parsing perf args")
			return nil
		} else {
			baseArgs := []string{
				"record",
				"-p", strconv.Itoa(self.job.Process.Pid),
				"-o", self.metadata.MetadataFilePath(journaledFiles[0]),
			}
			cmd = exec.Command("perf", append(baseArgs, args...)...)
		}
	} else if pc := self.jobInfo.ProfileConfig; pc == nil || pc.Command == "" {
		return nil
	} else {
		cmd = exec.Command(pc.Command, pc.ExpandedArgs(
			self.metadata.MetadataFilePath(core.PerfData),
			self.metadata.MetadataFilePath(core.ProfileOut),
			self.job.Process.Pid)...)
	}
	cmd.SysProcAttr = util.Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGINT)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	} else {
		perfDone := make(chan struct{})
		self.perfDone = perfDone
		go func(cmd *exec.Cmd, c chan<- struct{}) {
			cmd.Wait()
			close(c)
		}(cmd, perfDone)
		for _, file := range journaledFiles {
			self.metadata.UpdateJournal(file)
		}
		return nil
	}
}

func (self *runner) HandleSignal(sig os.Signal) {
	util.PrintInfo("monitor", "Caught signal %v", sig)
	cmd := self.job
	if cmd != nil {
		proc := cmd.Process
		if proc != nil {
			proc.Kill()
		}
	}
	if c := self.isDone; c != nil {
		// Wait up to 5 seconds for the child process to terminate.
		select {
		case <-time.After(time.Second * 5):
		case <-c:
		}
	}
	self.done()
	if err := self.metadata.WriteRaw(core.Errors, fmt.Sprintf("Caught signal %v", sig)); err != nil {
		util.PrintError(err, "monitor", "Could not write errors file.")
	}
	if err := self.metadata.UpdateJournal(core.Errors); err != nil {
		util.PrintError(err, "monitor", "Could not update error journal file.")
	}
}

// Reads at most n bytes from the reader, returning when either n bytes are read
// or the end of the reader is reached.  Errors are ignored.
func readBytes(n int, reader *os.File) []byte {
	if n <= 0 {
		panic("Cannot read non-positive number of bytes!")
	}
	result := make([]byte, n)
	cursor := 0
	for {
		lastRead, err := reader.Read(result[cursor:])
		if lastRead > 0 {
			cursor += lastRead
		}
		if err != nil || lastRead <= 0 || cursor >= n {
			return result[:cursor]
		}
	}
}

// This error contains a string written by the stage code.  It is already
// formatted, so when this is seen, any additional message is ignored.
type stageReturnedError struct {
	message string
}

func (self *stageReturnedError) Error() string {
	return self.message
}

// Returns true if sig is a signal which we expect is not due to a
// bug in the stage code.
func externalSignal(sig syscall.Signal) bool {
	for _, handled := range util.HANDLED_SIGNALS {
		if sig == handled {
			return true
		}
	}
	// SIGKLL isn't in the handled set because it can't be handled, but
	// should be treated equivalently to SIGTERM for these purposes.
	if sig == syscall.SIGKILL {
		return true
	}
	return false
}

// Convert an exec.ExitError to a stageReturnedError if the failure was due to
// one of the signals that we choose to handle.  This allows restart logic to
// work correctly.
func sigToErr(err error) error {
	if err == nil {
		return err
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if state, ok := exitErr.Sys().(*syscall.WaitStatus); ok &&
			state.Signaled() && externalSignal(state.Signal()) {
			return &stageReturnedError{message: fmt.Sprintf("signal: %v", state.Signal())}
		}
	}
	return err
}

// Wait for the process to complete or, if monitoring is enabled, for it to
// exceed its memory quota.
func (self *runner) WaitLoop() {
	wait := make(chan error, 1)
	go func() {
		errorBytes := readBytes(8100, self.errorReader)
		if len(errorBytes) > 0 {
			// If the job has finished, we want to wait on it so it isn't
			// a zombie while we do our cleanup, and also so that its rusage
			// gets reported.  However, if it doesn't die we don't want to
			// block our own exit.  Under most circumstances the process will
			// have already terminated by the time we get here.
			go func() {
				self.job.Wait()
				close(self.isDone)
			}()
			wait <- &stageReturnedError{message: string(errorBytes)}
		} else {
			close(self.isDone)
			wait <- sigToErr(self.job.Wait())
		}
	}()
	// Make sure we record at least one memory high-water mark, even
	// for short stages.
	self.getChildMemGB()
	lastHeartbeat := time.Now()
	err := func() error {
		defer self.errorReader.Close()
		timer := time.NewTimer(MemorySampleInterval)
		for {
			select {
			case err := <-wait:
				return err
			case <-timer.C:
				if err := self.monitor(&lastHeartbeat); err != nil {
					return err
				}
				timer.Reset(MemorySampleInterval)
			}
		}
	}()
	{
		// Wait up to 5 seconds for the job to finish, to ensure we get rusage.
		select {
		case <-time.After(time.Second * 5):
			self.job.Process.Signal(syscall.SIGKILL)
		case <-self.isDone:
		}
	}
	util.EnterCriticalSection()
	defer util.ExitCriticalSection()
	if err != nil {
		self.Fail(err, "Job failed in stage code")
	} else {
		self.Complete()
	}
}

func (self *runner) getChildMemGB() float64 {
	proc := self.job.Process
	if proc == nil {
		return 0
	}
	io := make(map[int]*core.IoAmount)
	mem, err := core.GetProcessTreeMemory(proc.Pid, true, io)
	mem.IncreaseRusage(core.GetRusage())
	self.highMem.IncreaseTo(mem)
	if err != nil {
		util.LogError(err, "monitor", "Error updating job statistics.")
	} else {
		self.ioStats.Update(io, time.Now())
	}
	return float64(mem.Rss) / (1024 * 1024 * 1024)
}

func (self *runner) monitor(lastHeartbeat *time.Time) error {
	if mem := self.getChildMemGB(); mem > float64(self.jobInfo.MemGB) {
		if self.jobInfo.Monitor == "monitor" {
			self.job.Process.Kill()
			return fmt.Errorf("Stage exceeded its memory quota (using %.1f, allowed %dG)",
				mem, self.jobInfo.MemGB)
		} else {
			util.LogInfo("monitor",
				"Stage exceeded its memory quota (using %.1f, allowed %dG)",
				mem, self.jobInfo.MemGB)
		}
	}
	if time.Since(*lastHeartbeat) > HeartbeatInterval {
		if err := self.metadata.UpdateJournal(core.Heartbeat); err != nil {
			util.PrintError(err, "monitor", "Could not write heartbeat.")
		} else {
			*lastHeartbeat = time.Now()
		}
	}
	return nil
}
