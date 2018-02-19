// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

package core

// Code to figure out physical core count.

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

func GetCPUInfo() (int, int, int, int) {
	// From Linux /proc/cpuinfo, count sockets, physical cores, and logical cores.
	// runtime.numCPU() does not provide this
	//
	fname := "/proc/cpuinfo"

	f, err := os.Open(fname)
	if err != nil {
		return 0, 0, 0, 0
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	sockets := -1
	physicalCoresPerSocket := -1
	logicalCores := -1

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}

		fields := strings.Split(string(line), ":")
		if len(fields) < 2 {
			continue
		}

		k := strings.TrimSpace(fields[0])
		v := strings.TrimSpace(fields[1])

		switch k {
		case "physical id":
			i, err := strconv.ParseInt(v, 10, 32)
			if err == nil && int(i) > sockets {
				sockets = int(i)
			}
		case "core id":
			i, err := strconv.ParseInt(v, 10, 32)
			if err == nil && int(i) > physicalCoresPerSocket {
				physicalCoresPerSocket = int(i)
			}
		case "processor":
			i, err := strconv.ParseInt(v, 10, 32)
			if err == nil && int(i) > logicalCores {
				logicalCores = int(i)
			}
		}
	}

	if sockets > -1 && physicalCoresPerSocket > -1 && logicalCores > -1 {
		sockets += 1
		physicalCoresPerSocket += 1
		physicalCores := sockets * physicalCoresPerSocket
		logicalCores += 1
		return sockets, physicalCoresPerSocket, physicalCores, logicalCores
	} else {
		return 0, 0, 0, 0
	}
}
