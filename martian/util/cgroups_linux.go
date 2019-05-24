// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// Code for querying cgroup information.

package util

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"path"
)

// Find out where the memory cgroup controller is mounted.
func findCgroupMount(cgType []byte) string {
	m, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer m.Close()
	scanner := bufio.NewScanner(m)
	for scanner.Scan() {
		fields := bytes.Fields(scanner.Bytes())
		if len(fields) >= 10 && string(fields[8]) == "cgroup" && bytes.Contains(fields[9], cgType) {
			return string(fields[4])
		}
	}
	return ""
}

func findCgroup(cgType []byte) string {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := bytes.SplitN(scanner.Bytes(), []byte{':'}, 3)
		if len(fields) >= 3 && bytes.Contains(fields[1], cgType) {
			return string(bytes.TrimSpace(fields[2]))
		}
	}
	return ""
}

func getCgroupPath(cgType []byte) string {
	root := findCgroupMount(cgType)
	if root == "" {
		return ""
	}
	p := findCgroup(cgType)
	if p == "" {
		return ""
	}
	return root + p
}

func getMemoryCgroupPath() string {
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
	p := getMemoryCgroupPath()
	if p == "" {
		return 0, 0, 0
	}
	limit, usage = readMemoryStat(p)
	for _, name := range [...]string{
		"memory.limit_in_bytes",
		"memory.memsw.limit_in_bytes",
	} {
		if b, err := ioutil.ReadFile(path.Join(p, name)); err == nil {
			if v := parseCgroupInt(b); v != 0 {
				if v < limit || limit == 0 {
					limit = v
				}
			}
		}
	}
	if b, err := ioutil.ReadFile(path.Join(p,
		"memory.soft_limit_in_bytes")); err == nil {
		softLimit = parseCgroupInt(b)
	}
	return limit, softLimit, usage
}
