// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// Code for querying cgroup information.

package util

import (
	"bufio"
	"bytes"
	"os"
	"path"
)

// These are effectively constants, but can be overridden for unit tests.
var (
	mountInfoPath  = "/proc/self/mountinfo"
	cgroupProcPath = "/proc/self/cgroup"
)

// Parse mount type and options from mountinfo line.
func getCgMountType(fields [][]byte) ([]byte, []byte) {
	for i, f := range fields {
		if len(f) == 1 && f[0] == '-' {
			if len(fields) < i+4 {
				return nil, nil
			}
			return fields[i+1], fields[i+3]
		}
	}
	return nil, nil
}

// Get the cgroup types from the mount options of a mount.
func getCgTypes(fields [][]byte) []byte {
	ty, opts := getCgMountType(fields)
	if string(ty) != "cgroup" {
		return nil
	}
	return opts
}

// Returns true if the mount type is cgroup2.
func isCgV2(fields [][]byte) bool {
	ty, _ := getCgMountType(fields)
	return string(ty) == "cgroup2"
}

// Find out where the memory cgroup controller is mounted.
func findCgroupMount(cgType []byte) string {
	m, err := os.Open(mountInfoPath)
	if err != nil {
		return ""
	}
	defer m.Close()
	scanner := bufio.NewScanner(m)
	for scanner.Scan() {
		fields := bytes.Fields(scanner.Bytes())
		if len(fields) >= 10 {
			if len(cgType) == 0 {
				if isCgV2(fields[6:]) {
					return string(fields[4])
				}
			} else if bytes.Contains(getCgTypes(fields[6:]), cgType) {
				return string(fields[4])
			}
		}
	}
	return ""
}

func findCgroup(cgType []byte) (string, bool) {
	f, err := os.Open(cgroupProcPath)
	if err != nil {
		return "", false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := bytes.SplitN(scanner.Bytes(), []byte{':'}, 3)
		if len(fields) >= 3 {
			if len(fields[1]) == 0 {
				return string(bytes.TrimSpace(fields[2])), true
			}
			if bytes.Contains(fields[1], cgType) {
				return string(bytes.TrimSpace(fields[2])), false
			}
		}
	}
	return "", false
}

func getCgroupPath(cgType []byte) (string, bool) {
	p, v2 := findCgroup(cgType)
	if p == "" {
		return "", false
	}
	if v2 {
		cgType = nil
	}
	root := findCgroupMount(cgType)
	if root == "" {
		return "", v2
	}
	return root + p, v2
}

func getMemoryCgroupPath() (string, bool) {
	return getCgroupPath([]byte("memory"))
}

func parseCgroupInt(b []byte) int64 {
	b = bytes.TrimSpace(b)
	if len(b) < 1 || b[0] < '0' || b[0] > '9' {
		// fast-path because cgroup limits don't have a sign prefix.
		return 0
	}
	result := int64(b[0] - '0')
	for _, r := range b[1:] {
		if r < '0' || r > '9' {
			return 0
		}
		result = 10*result + int64(r-'0')
	}
	return result
}

func readMemoryStat(p string) (limit, usage int64) {
	f, err := os.Open(path.Join(p, "memory.stat"))
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := bytes.Fields(scanner.Bytes())
		if len(fields) >= 2 {
			switch string(fields[0]) {
			case "hierarchical_memory_limit",
				"hierarchical_memsw_limit":
				if v := parseCgroupInt(fields[1]); v != 0 {
					if v < limit || limit == 0 {
						limit = v
					}
				}
			case "rss":
				usage = parseCgroupInt(fields[1])
				if usage < 0 {
					usage = 0
				}
			}
		}
	}
	return limit, usage
}

// Get the lowest rss memory limit in cgroups, as well as the current
// usage.
func GetCgroupMemoryLimit() (limit, softLimit, usage int64) {
	p, v2 := getMemoryCgroupPath()
	if p == "" {
		return 0, 0, 0
	}
	limit, usage = readMemoryStat(p)
	var limitFiles [2]string
	soft := "memory.soft_limit_in_bytes"
	if v2 {
		limitFiles = [...]string{
			"memory.max",
			"memory.swap.max",
		}
		soft = "memory.high"
	} else {
		limitFiles = [...]string{
			"memory.limit_in_bytes",
			"memory.memsw.limit_in_bytes",
		}
	}
	for _, name := range limitFiles {
		if b, err := os.ReadFile(path.Join(p, name)); err == nil {
			if v := parseCgroupInt(b); v != 0 {
				if v < limit || limit == 0 {
					limit = v
				}
			}
		}
	}
	if b, err := os.ReadFile(path.Join(p,
		soft)); err == nil {
		softLimit = parseCgroupInt(b)
	}
	return limit, softLimit, usage
}
