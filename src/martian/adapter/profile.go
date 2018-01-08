package adapter

import (
	"martian/core"
	"martian/util"
	"os"
	"runtime"
	"runtime/pprof"
)

func openMemProfile(metadata *core.Metadata) *os.File {
	if profDest, err := os.OpenFile(
		metadata.MetadataFilePath("profile_mem.pprof"),
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0666); err != nil {
		util.LogError(err, "adapter", "Could not open destination for memory profile.")
		return nil
	} else {
		return profDest
	}
}

func writeMemProfile(dest *os.File) {
	defer dest.Close()
	runtime.GC()
	if p := pprof.Lookup("heap"); p == nil {
		util.LogInfo("adapter", "No heap profile found.")
	} else if err := p.WriteTo(dest, 1); err != nil {
		util.LogError(err, "adapter", "Error writing heap profile.")
	}
}

func openCpuProfile(metadata *core.Metadata) *os.File {
	if profDest, err := os.OpenFile(
		metadata.MetadataFilePath("profile_cpu.pprof"),
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0666); err != nil {
		util.LogError(err, "adapter", "Could not open destination for cpu profile.")
		return nil
	} else {
		runtime.GC()
		if err := pprof.StartCPUProfile(profDest); err != nil {
			util.LogError(err, "adapter", "Could not start CPU profiling.")
			profDest.Close()
			return nil
		}
		return profDest
	}
}

func writeCpuProfile(f *os.File) {
	defer f.Close()
	pprof.StopCPUProfile()
}

func profileSplit(split SplitFunc) SplitFunc {
	if jobinfo.ProfileMode == core.MemProfile {
		return func(metadata *core.Metadata) (*core.StageDefs, error) {
			if profDest := openMemProfile(metadata); profDest != nil {
				defer writeMemProfile(profDest)
			}
			runtime.GC()
			return split(metadata)
		}
	} else if jobinfo.ProfileMode == core.CpuProfile ||
		jobinfo.ProfileMode == core.LineProfile {
		return func(metadata *core.Metadata) (*core.StageDefs, error) {
			if profDest := openCpuProfile(metadata); profDest != nil {
				defer writeCpuProfile(profDest)
			}
			return split(metadata)
		}
	} else {
		return split
	}
}

func profileMain(main MainFunc) MainFunc {
	if jobinfo.ProfileMode == core.MemProfile {
		return func(metadata *core.Metadata) (interface{}, error) {
			if profDest := openMemProfile(metadata); profDest != nil {
				defer writeMemProfile(profDest)
			}
			runtime.GC()
			return main(metadata)
		}
	} else if jobinfo.ProfileMode == core.CpuProfile ||
		jobinfo.ProfileMode == core.LineProfile {
		return func(metadata *core.Metadata) (interface{}, error) {
			if profDest := openCpuProfile(metadata); profDest != nil {
				defer writeCpuProfile(profDest)
			}
			return main(metadata)
		}
	} else {
		return main
	}
}
