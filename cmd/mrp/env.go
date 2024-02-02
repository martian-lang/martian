//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

// Environment logging.

package main

import (
	"runtime"
	"strings"

	"github.com/martian-lang/martian/martian/util"
)

// List of environment variables which might be useful in debugging.
func loggedEnvs(s string) bool {
	switch s {
	case "_CONDA_PYTHON_SYSCONFIGDATA_NAME",
		"_PYTHON_SYSCONFIGDATA_NAME",
		"BATCH_SYSTEM", // HTCondor
		"COMMD_PORT",
		"CWD",
		"ENVIRONMENT", // SGE
		"EXE",
		"GODEBUG",
		"GOMAXPROCS",
		"HOME",
		"HOST",
		"HOSTNAME",
		"HOSTTYPE", // LSF
		"HYDRA_ROOT",
		"JPY_PARENT_PID", // Set inside jupyter notebooks
		"LANG",
		"LIBRARY_PATH",
		"LOGNAME",
		"MPLCONFIGDIR",
		"NHOSTS",  // SGE
		"NQUEUES", // SGE
		"NSLOTS",  // SGE
		"PATH",
		"PID",
		"PWD",
		"SHELL",
		"SHLVL",
		"SPOOLDIR", // LSF
		"TERM",
		"TMPDIR",
		"USER",
		"WAFDIR",
		"_":
		return true
	}
	return false
}

// List of environment variable prefixes which might be useful in debugging.
// These are accepted for variables of the form "KEY_*"
func loggedEnvPrefix(s string) bool {
	switch s {
	case "BASH",
		"CONDA",
		"_CONDOR", // HT Condor
		"CUDA",
		"DYLD", // Linker
		"EC2",
		"EGO", // LSF
		"HDF5",
		"JAVA",
		"JOB", // SGE
		"LC",
		"LD",     // Linker
		"LS",     // LSF
		"LSB",    // LSF
		"LSF",    // LSF
		"MRO",    // Martian
		"MALLOC", // jemalloc
		"MARTIAN",
		"MKL",
		"MYSYS2", // Anaconda
		"NUMEXPR",
		"OMP",
		"PBS", // PBS
		"PD",
		"RUST",
		"SBATCH",  // Slurm
		"SELINUX", // Linux
		"SGE",
		"SLURM",
		"SSH",
		"TENX",
		"XDG":
		return true
	}
	return false
}

// Returns true if the environment variable should be logged.
func logEnv(env string) bool {
	if loggedEnvs(env) {
		return true
	}
	// Various important PYTHON environment variables don't have a _ separator.
	if strings.HasPrefix(env, "PYTHON") {
		return true
	}
	if idx := strings.Index(env, "_"); idx > 0 {
		env = env[:idx]
	}
	return loggedEnvPrefix(env)
}

// Log startup parameters.
func logEnviron(ver string, args, envs []string, pid int) {
	util.Println("Martian Runtime - %s", ver)
	util.LogInfo("build  ", "Built with Go version %s", runtime.Version())
	util.LogInfo("cmdline", "%s", strings.Join(args, " "))
	util.LogInfo("pid    ", "%d", pid)

	for _, env := range envs {
		i := strings.IndexRune(env, '=')
		if i > 0 && logEnv(env[:i]) {
			util.LogInfo("environ", "%s", env)
		}
	}
}
