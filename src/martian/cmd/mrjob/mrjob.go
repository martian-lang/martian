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
	"martian/core"
	"martian/util"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type runner struct {
	job         *exec.Cmd
	log         *os.File
	errorReader *os.File
	highMem     core.ObservedMemory
	metadata    *core.Metadata
	jobInfo     *core.JobInfo
	start       time.Time
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
		metadata: core.NewMetadataRunWithJournalPath(fqname, metadataPath, filesPath, journalPath, runType),
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
	var rlim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
		util.PrintError(err, "monitor", "Error getting file rlimit.")
		return
	}
	rlim.Cur = rlim.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
		util.PrintError(err, "monitor",
			"Could not increase the file handle ulimit to %d", rlim.Max)
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
		fmt.Fprintf(os.Stderr, errStr)
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
		util.PrintError(jErr, "monitor", "Could not update error journal file.")
	}
	os.Exit(0)
}

func (self *runner) Complete() {
	self.done()
	if writeError := self.metadata.WriteTime(core.CompleteFile); writeError != nil {
		util.PrintError(writeError, "monitor", "Could not write complete file.")
	}
	if jErr := self.metadata.UpdateJournal(core.CompleteFile); jErr != nil {
		util.PrintError(jErr, "monitor", "Could not update complete journal file.")
	}
	os.Exit(0)
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
	cmd.SysProcAttr = core.Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGKILL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
	switch self.jobInfo.ProfileMode {
	case core.PyflameProfile:
		journaledFiles = []core.MetadataFileName{
			"profile.out",
			"profile.out.html",
		}
		cmd = exec.Command("pyflame",
			"-s", "-1",
			"-o", self.metadata.MetadataFilePath(journaledFiles[0]),
			"-H", self.metadata.MetadataFilePath(journaledFiles[1]),
			strconv.Itoa(self.job.Process.Pid),
		)
	case core.PerfRecordProfile:
		events := os.Getenv("MRO_PERF_EVENTS")
		if events == "" {
			events = "task-clock,bpf-output"
		}
		freq := os.Getenv("MRO_PERF_FREQ")
		if freq == "" {
			freq = "200"
		}
		duration := os.Getenv("MRO_PERF_DURATION")
		if duration == "" {
			duration = "2400"
		}
		journaledFiles = []core.MetadataFileName{"perf.data"}
		// Running perf record for 2400 seconds (40 minutes) with these default
		// settings will produce about 26MB per thread/process.
		cmd = exec.Command("perf",
			"record", "-g", "-F", freq,
			"-o", self.metadata.MetadataFilePath(journaledFiles[0]),
			"-e", events,
			"-p", strconv.Itoa(self.job.Process.Pid),
			"sleep", duration,
		)
	default:
		return nil
	}
	cmd.SysProcAttr = core.Pdeathsig(&syscall.SysProcAttr{}, syscall.SIGINT)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	} else {
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

func (self *runner) WaitLoop() {
	wait := make(chan error)
	go func() {
		errorBytes := readBytes(8100, self.errorReader)
		if len(errorBytes) > 0 {
			wait <- &stageReturnedError{message: string(errorBytes)}
		} else {
			wait <- self.job.Wait()
		}
	}()
	// Make sure we record at least one memory high-water mark, even
	// for short stages.
	self.getChildMemGB()
	err := func() error {
		defer self.errorReader.Close()
		timer := time.NewTimer(time.Second * 120)
		for {
			select {
			case err := <-wait:
				return err
			case <-timer.C:
				if err := self.monitor(); err != nil {
					return err
				}
				timer.Reset(time.Second * 120)
			}
		}
	}()
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
	mem, _ := core.GetProcessTreeMemory(proc.Pid, true)
	mem.IncreaseRusage(core.GetRusage())
	self.highMem.IncreaseTo(mem)
	return float64(mem.Rss) / (1024 * 1024 * 1024)
}

func (self *runner) monitor() error {
	if self.jobInfo.Monitor == "monitor" {
		if mem := self.getChildMemGB(); mem > float64(self.jobInfo.MemGB) {
			self.job.Process.Kill()
			return fmt.Errorf("Stage exceeded its memory quota (using %.1f, allowed %dG)", mem, self.jobInfo.MemGB)
		}
	}
	if err := self.metadata.UpdateJournal(core.Heartbeat); err != nil {
		util.PrintError(err, "monitor", "Could not write heartbeat.")
	}
	return nil
}
