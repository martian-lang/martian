// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package core

// Data structures for managing stage performance information.
// Functions:
// - Reduce jobinfo to important metrics
// - Compute aggregational stats multiple jobinfos
// - Get arguments and compute file sizes (if they exist)

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/martian-lang/martian/martian/syntax"
	"github.com/martian-lang/martian/martian/util"
)

type RusageInfo struct {
	Self     *Rusage `json:"self,omitempty"`
	Children *Rusage `json:"children,omitempty"`
}

type Rusage struct {
	MaxRss       int     `json:"ru_maxrss"`
	SharedRss    int     `json:"ru_ixrss"`
	UnsharedRss  int     `json:"ru_idrss"`
	MinorFaults  int     `json:"ru_minflt"`
	MajorFaults  int     `json:"ru_majflt"`
	SwapOuts     int     `json:"ru_nswap"`
	UserTime     float64 `json:"ru_utime"`
	SystemTime   float64 `json:"ru_stime"`
	InBlocks     int     `json:"ru_inblock"`
	OutBlocks    int     `json:"ru_oublock"`
	MessagesSent int     `json:"ru_msgsnd"`
	MessagesRcvd int     `json:"ru_msgrcv"`
	SignalsRcvd  int     `json:"ru_nsignals"`
	CtxSwitches  int     `json:"ru_nivcsw"`
}

// Current observed memory usage.
type ObservedMemory struct {
	Rss    int64 `json:"rss"`
	Shared int64 `json:"shared"`
	Vmem   int64 `json:"vmem"`
	Text   int64 `json:"text"`
	Stack  int64 `json:"stack"`
	Procs  int   `json:"proc_count"`
}

// Increase this value to max(this,other).
func (self *ObservedMemory) IncreaseTo(other ObservedMemory) {
	if other.Rss > self.Rss {
		self.Rss = other.Rss
	}
	if other.Vmem > self.Vmem {
		self.Vmem = other.Vmem
	}
	if other.Shared > self.Shared {
		self.Shared = other.Shared
	}
	if other.Text > self.Text {
		self.Text = other.Text
	}
	if other.Stack > self.Stack {
		self.Stack = other.Stack
	}
	if other.Procs > self.Procs {
		self.Procs = other.Procs
	}
}

// Add other to this.
func (self *ObservedMemory) Add(other ObservedMemory) {
	self.Rss += other.Rss
	self.Vmem += other.Vmem
	self.Shared += other.Shared
	self.Text += other.Text
	self.Stack += other.Stack
	self.Procs += other.Procs
}

// Increase this value to the max RSS reported by getrusage, if it
// is higher.
func (self *ObservedMemory) IncreaseRusage(other *RusageInfo) {
	if other == nil {
		return
	}
	if other.Self != nil {
		oRss := int64(other.Self.MaxRss) * 1024
		if oRss > self.Rss {
			self.Rss = oRss
		}
	}
	if other.Children != nil {
		oRss := int64(other.Children.MaxRss) * 1024
		if oRss > self.Rss {
			self.Rss = oRss
		}
	}
}

func (self *ObservedMemory) IsZero() bool {
	return self.Rss == 0 && self.Vmem == 0 &&
		self.Shared == 0 && self.Text == 0 &&
		self.Stack == 0
}

func (self *ObservedMemory) RssKb() int {
	return int((self.Rss + 512) / 1024)
}

func (self *ObservedMemory) VmemKb() int {
	return int((self.Vmem + 512) / 1024)
}

type ProcessStats struct {
	Pid    int
	Memory ObservedMemory
	IO     IoAmount
	// The command executed for this process
	Cmdline []string
	// The depth in the process tree.
	Depth int
}

type ProcessTree []ProcessStats

func (tree ProcessTree) Format(indent string) string {
	if len(tree) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(indent)
	builder.WriteString(
		"   PID VSZ(mb) RSS(mb) Procs  Read(mb) (calls) Write(mb) (calls) COMMAND\n")
	for i, proc := range tree {
		if i > 0 {
			builder.WriteRune('\n')
		}
		builder.WriteString(indent)
		fmt.Fprintf(&builder, "%6d %7.f %7.f %5d %9.f %7d %9.f %7d",
			proc.Pid,
			float64(proc.Memory.Vmem)/(1024*1024),
			float64(proc.Memory.Rss)/(1024*1024),
			proc.Memory.Procs,
			float64(proc.IO.Read.BlockBytes)/(1024*1024),
			proc.IO.Read.Syscalls,
			float64(proc.IO.Write.BlockBytes)/(1024*1024),
			proc.IO.Write.Syscalls)
		for i := 0; i < proc.Depth; i++ {
			builder.WriteString("  ")
		}
		for _, arg := range proc.Cmdline {
			builder.WriteRune(' ')
			builder.WriteString(arg)
		}
	}
	return builder.String()
}

type PerfInfo struct {
	NumJobs         int       `json:"num_jobs"`
	NumThreads      int       `json:"num_threads"`
	Duration        float64   `json:"duration"`
	CoreHours       float64   `json:"core_hours"`
	MaxRss          int       `json:"maxrss"`
	MaxVmem         int       `json:"maxvmem"`
	InBlocks        int       `json:"in_blocks"`
	OutBlocks       int       `json:"out_blocks"`
	TotalBlocks     int       `json:"total_blocks"`
	InBlocksRate    float64   `json:"in_blocks_rate"`
	OutBlocksRate   float64   `json:"out_blocks_rate"`
	TotalBlocksRate float64   `json:"total_blocks_rate"`
	InBytes         int64     `json:"in_bytes"`
	OutBytes        int64     `json:"out_bytes"`
	InBytesRate     float64   `json:"in_bytes_rate"`
	OutBytesRate    float64   `json:"out_bytes_rate"`
	InBytesPeak     float64   `json:"in_bytes_peak"`
	OutBytesPeak    float64   `json:"out_bytes_peak"`
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	WallTime        float64   `json:"walltime"`
	UserTime        float64   `json:"usertime"`
	SystemTime      float64   `json:"systemtime"`
	TotalFiles      uint      `json:"total_files"`
	TotalBytes      uint64    `json:"total_bytes"`
	OutputFiles     uint      `json:"output_files"`
	OutputBytes     uint64    `json:"output_bytes"`
	VdrFiles        uint      `json:"vdr_files"`
	VdrBytes        uint64    `json:"vdr_bytes"`

	// Deviation for a single job is deviation over time as measured by mrjob.
	// For node aggregates, it's the deviation between child nodes.
	InBytesDev  float64 `json:"in_bytes_dev"`
	OutBytesDev float64 `json:"out_bytes_dev"`
}

type ChunkPerfInfo struct {
	Index      int       `json:"index"`
	ChunkStats *PerfInfo `json:"chunk_stats"`
}

type StagePerfInfo struct {
	Name   string `json:"name"`
	Fqname string `json:"fqname"`
	Forki  int    `json:"forki"`
}

type ForkPerfInfo struct {
	Stages     []*StagePerfInfo `json:"stages"`
	Index      int              `json:"index"`
	Chunks     []*ChunkPerfInfo `json:"chunks"`
	SplitStats *PerfInfo        `json:"split_stats"`
	JoinStats  *PerfInfo        `json:"join_stats"`
	ForkStats  *PerfInfo        `json:"fork_stats"`
}

type NodeByteStamp struct {
	Timestamp   time.Time `json:"ts"`
	Bytes       int64     `json:"bytes"`
	Description string    `json:"desc"`
}

type NodePerfInfo struct {
	Name      string                   `json:"name"`
	Fqname    string                   `json:"fqname"`
	Type      syntax.CallGraphNodeType `json:"type"`
	Forks     []*ForkPerfInfo          `json:"forks"`
	MaxBytes  int64                    `json:"maxbytes"`
	BytesHist []*NodeByteStamp         `json:"bytehist"`
	HighMem   *ObservedMemory          `json:"highmem,omitempty"`
}

func reduceJobInfo(jobInfo *JobInfo, outputPaths []string, numThreads int) *PerfInfo {
	perfInfo := PerfInfo{}
	timeLayout := "2006-01-02 15:04:05"

	perfInfo.NumJobs = 1
	perfInfo.NumThreads = numThreads
	if jobInfo.WallClockInfo != nil {
		perfInfo.Start, _ = time.Parse(timeLayout, jobInfo.WallClockInfo.Start)
		perfInfo.End, _ = time.Parse(timeLayout, jobInfo.WallClockInfo.End)
		perfInfo.Duration = jobInfo.WallClockInfo.Duration
		perfInfo.WallTime = perfInfo.End.Sub(perfInfo.Start).Seconds()
	}
	if jobInfo.RusageInfo != nil {
		self := jobInfo.RusageInfo.Self
		children := jobInfo.RusageInfo.Children

		perfInfo.CoreHours = float64(perfInfo.NumThreads) * perfInfo.Duration / 3600.0
		perfInfo.MaxRss = max(self.MaxRss, children.MaxRss)
		perfInfo.InBlocks = self.InBlocks + children.InBlocks
		perfInfo.OutBlocks = self.OutBlocks + children.OutBlocks
		perfInfo.TotalBlocks = perfInfo.InBlocks + perfInfo.OutBlocks
		perfInfo.UserTime = self.UserTime + children.UserTime
		perfInfo.SystemTime = self.SystemTime + children.SystemTime
		if perfInfo.Duration > 0 {
			perfInfo.InBlocksRate = float64(perfInfo.InBlocks) / perfInfo.Duration
			perfInfo.OutBlocksRate = float64(perfInfo.OutBlocks) / perfInfo.Duration
			perfInfo.TotalBlocksRate = float64(perfInfo.TotalBlocks) / perfInfo.Duration
		}
	}
	if jobInfo.MemoryUsage != nil {
		if perfInfo.MaxRss < jobInfo.MemoryUsage.RssKb() {
			perfInfo.MaxRss = jobInfo.MemoryUsage.RssKb()
		}
		perfInfo.MaxVmem = jobInfo.MemoryUsage.VmemKb()
	}
	if jobInfo.IoStats != nil {
		perfInfo.InBytes = jobInfo.IoStats.Total.Read.BlockBytes
		perfInfo.OutBytes = jobInfo.IoStats.Total.Write.BlockBytes
		if perfInfo.Duration > 0 {
			perfInfo.InBytesRate = float64(perfInfo.InBytes) / perfInfo.Duration
			perfInfo.OutBytesRate = float64(perfInfo.OutBytes) / perfInfo.Duration
		}
		perfInfo.InBytesPeak = jobInfo.IoStats.RateMax.Read.BlockBytes
		perfInfo.OutBytesPeak = jobInfo.IoStats.RateMax.Write.BlockBytes
		perfInfo.InBytesDev = jobInfo.IoStats.RateDev.Read.BlockBytes
		perfInfo.OutBytesDev = jobInfo.IoStats.RateDev.Write.BlockBytes
	}

	perfInfo.OutputFiles, perfInfo.OutputBytes = util.GetDirectorySize(outputPaths)
	perfInfo.TotalFiles = perfInfo.OutputFiles
	perfInfo.TotalBytes = perfInfo.OutputBytes

	return &perfInfo
}

func ComputeStats(perfInfos []*PerfInfo, outputPaths []string, vdrKillReport *VDRKillReport) *PerfInfo {
	aggPerfInfo := &PerfInfo{}
	fmax := func(x, y float64) float64 {
		if x > y {
			return x
		} else {
			return y
		}
	}
	square := func(x float64) float64 {
		return x * x
	}
	for _, perfInfo := range perfInfos {
		if aggPerfInfo.Start.IsZero() || (!perfInfo.Start.IsZero() && aggPerfInfo.Start.After(perfInfo.Start)) {
			aggPerfInfo.Start = perfInfo.Start
		}
		if aggPerfInfo.End.Before(perfInfo.End) {
			aggPerfInfo.End = perfInfo.End
		}

		aggPerfInfo.NumJobs += perfInfo.NumJobs
		aggPerfInfo.NumThreads += perfInfo.NumThreads
		aggPerfInfo.Duration += perfInfo.Duration
		aggPerfInfo.CoreHours += perfInfo.CoreHours
		aggPerfInfo.MaxRss = max(aggPerfInfo.MaxRss, perfInfo.MaxRss)
		aggPerfInfo.MaxVmem = max(aggPerfInfo.MaxVmem, perfInfo.MaxVmem)
		aggPerfInfo.OutBlocks += perfInfo.OutBlocks
		aggPerfInfo.InBlocks += perfInfo.InBlocks
		aggPerfInfo.TotalBlocks += perfInfo.TotalBlocks
		aggPerfInfo.OutBytes += perfInfo.OutBytes
		aggPerfInfo.InBytes += perfInfo.InBytes
		aggPerfInfo.OutBytesPeak = fmax(aggPerfInfo.OutBytesPeak, perfInfo.OutBytesPeak)
		aggPerfInfo.InBytesPeak = fmax(aggPerfInfo.InBytesPeak, perfInfo.InBytesPeak)
		aggPerfInfo.OutputFiles += perfInfo.OutputFiles
		aggPerfInfo.OutputBytes += perfInfo.OutputBytes
		aggPerfInfo.UserTime += perfInfo.UserTime
		aggPerfInfo.SystemTime += perfInfo.SystemTime

		if perfInfo.Duration > 0 {
			// Accumulate sum^2 bytes here.  Convert to deviation at the end.
			aggPerfInfo.InBytesDev += square(float64(perfInfo.InBytes)) / perfInfo.Duration
			aggPerfInfo.OutBytesDev += square(float64(perfInfo.OutBytes)) / perfInfo.Duration
		}

		if vdrKillReport == nil {
			// If VDR kill report is nil, use perf reports' VDR stats
			aggPerfInfo.VdrFiles += perfInfo.VdrFiles
			aggPerfInfo.VdrBytes += perfInfo.VdrBytes
		}
	}
	if aggPerfInfo.Duration > 0 {
		aggPerfInfo.InBlocksRate = float64(aggPerfInfo.InBlocks) / aggPerfInfo.Duration
		aggPerfInfo.OutBlocksRate = float64(aggPerfInfo.OutBlocks) / aggPerfInfo.Duration
		aggPerfInfo.TotalBlocksRate = float64(aggPerfInfo.TotalBlocks) / aggPerfInfo.Duration
		aggPerfInfo.InBytesRate = float64(aggPerfInfo.InBytes) / aggPerfInfo.Duration
		aggPerfInfo.OutBytesRate = float64(aggPerfInfo.OutBytes) / aggPerfInfo.Duration
		safeSqrt := func(x float64) float64 {
			if x > 0 {
				return math.Sqrt(x)
			} else {
				return 0
			}
		}
		aggPerfInfo.InBytesDev = safeSqrt(
			aggPerfInfo.InBytesDev/aggPerfInfo.Duration -
				aggPerfInfo.InBytesRate*aggPerfInfo.InBytesRate)
		aggPerfInfo.OutBytesDev = safeSqrt(
			aggPerfInfo.OutBytesDev/aggPerfInfo.Duration -
				aggPerfInfo.OutBytesRate*aggPerfInfo.OutBytesRate)
	}
	if vdrKillReport != nil {
		aggPerfInfo.VdrFiles = vdrKillReport.Count
		aggPerfInfo.VdrBytes = vdrKillReport.Size
	}
	aggPerfInfo.WallTime = aggPerfInfo.End.Sub(aggPerfInfo.Start).Seconds()
	outputFiles, outputBytes := util.GetDirectorySize(outputPaths)
	aggPerfInfo.OutputFiles += outputFiles
	aggPerfInfo.OutputBytes += outputBytes
	aggPerfInfo.TotalFiles = aggPerfInfo.OutputFiles + aggPerfInfo.VdrFiles
	aggPerfInfo.TotalBytes = aggPerfInfo.OutputBytes + aggPerfInfo.VdrBytes
	return aggPerfInfo
}
