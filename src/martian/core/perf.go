// Functions:
// - Reduce jobinfo to important metrics
// - Compute aggregational stats multiple jobinfos
// - Get arguments and compute file sizes (if they exist)
package core

import (
	"time"
)

type JobInfo struct {
	Name          string         `json:"name"`
	Pid           int            `json:"int"`
	Host          string         `json:"host"`
	Type          string         `json:"type"`
	Cwd           string         `json:"cwd"`
	PythonInfo    *PythonInfo    `json:"python"`
	RusageInfo    *RusageInfo    `json:"rusage"`
	WallClockInfo *WallClockInfo `json:"wallclock"`
}

type PythonInfo struct {
	BinPath string `json:"binpath"`
	Version string `json:"version"`
}

type RusageInfo struct {
	Self     *Rusage `json:"self"`
	Children *Rusage `json:"children"`
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
}

type WallClockInfo struct {
	Start    string  `json:"start"`
	End      string  `json:"end"`
	Duration float64 `json:"duration_seconds"`
}

type PerfInfo struct {
	NumJobs         int       `json:"num_jobs"`
	NumThreads      int       `json:"num_threads"`
	Duration        float64   `json:"duration"`
	CoreHours       float64   `json:"core_hours"`
	MaxRss          int       `json:"maxrss"`
	InBlocks        int       `json:"in_blocks"`
	OutBlocks       int       `json:"out_blocks"`
	TotalBlocks     int       `json:"total_blocks"`
	InBlocksRate    float64   `json:"in_blocks_rate"`
	OutBlocksRate   float64   `json:"out_blocks_rate"`
	TotalBlocksRate float64   `json:"total_blocks_rate"`
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
}

func (self *PerfInfo) markOutputAsVDR() {
	self.VdrBytes = self.OutputBytes
	self.VdrFiles = self.OutputFiles
	self.OutputBytes = 0
	self.OutputFiles = 0
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

type NodePerfInfo struct {
	Name   string          `json:"name"`
	Fqname string          `json:"fqname"`
	Type   string          `json:"type"`
	Forks  []*ForkPerfInfo `json:"forks"`
}

func reduceJobInfo(jobInfo *JobInfo, outputPaths []string, numThreads int) *PerfInfo {
	perfInfo := &PerfInfo{}
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

	perfInfo.OutputFiles, perfInfo.OutputBytes = GetDirectorySize(outputPaths)
	perfInfo.TotalFiles = perfInfo.OutputFiles
	perfInfo.TotalBytes = perfInfo.OutputBytes

	return perfInfo
}

func ComputeStats(perfInfos []*PerfInfo, outputPaths []string) *PerfInfo {
	aggPerfInfo := &PerfInfo{}
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
		aggPerfInfo.OutBlocks += perfInfo.OutBlocks
		aggPerfInfo.InBlocks += perfInfo.InBlocks
		aggPerfInfo.TotalBlocks += perfInfo.TotalBlocks
		aggPerfInfo.OutputFiles += perfInfo.OutputFiles
		aggPerfInfo.OutputBytes += perfInfo.OutputBytes
		aggPerfInfo.UserTime += perfInfo.UserTime
		aggPerfInfo.SystemTime += perfInfo.SystemTime
		aggPerfInfo.VdrFiles += perfInfo.VdrFiles
		aggPerfInfo.VdrBytes += perfInfo.VdrBytes
	}
	if aggPerfInfo.Duration > 0 {
		aggPerfInfo.InBlocksRate = float64(aggPerfInfo.InBlocks) / aggPerfInfo.Duration
		aggPerfInfo.OutBlocksRate = float64(aggPerfInfo.OutBlocks) / aggPerfInfo.Duration
		aggPerfInfo.TotalBlocksRate = float64(aggPerfInfo.TotalBlocks) / aggPerfInfo.Duration
	}
	aggPerfInfo.WallTime = aggPerfInfo.End.Sub(aggPerfInfo.Start).Seconds()
	outputFiles, outputBytes := GetDirectorySize(outputPaths)
	aggPerfInfo.OutputFiles += outputFiles
	aggPerfInfo.OutputBytes += outputBytes
	aggPerfInfo.TotalFiles = aggPerfInfo.OutputFiles + aggPerfInfo.VdrFiles
	aggPerfInfo.TotalBytes = aggPerfInfo.OutputBytes + aggPerfInfo.VdrBytes
	return aggPerfInfo
}
