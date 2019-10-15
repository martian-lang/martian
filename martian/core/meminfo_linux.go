// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

//  Utility method to parse /proc/meminfo

package core

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
)

type MemInfo struct {
	Total      int64
	Used       int64
	Free       int64
	ActualFree int64
	ActualUsed int64
	available  int64
	buffers    int64
	cached     int64
}

func (m *MemInfo) Get() error {
	m.available = -1
	m.buffers = 0
	m.cached = 0

	err := m.parseMeminfo()

	if m.available == -1 {
		m.ActualFree = m.Free + m.buffers + m.cached
	} else {
		m.ActualFree = m.available
	}

	m.Used = m.Total - m.Free
	m.ActualUsed = m.Total - m.ActualFree
	return err
}

func (m *MemInfo) parseMeminfo() error {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		b := scanner.Bytes()
		nameLen := bytes.IndexRune(b, ':')
		if nameLen <= 0 {
			continue
		}
		if err := m.parseMeminfoLine(b[:nameLen], b[nameLen+1:]); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func parseMeminfoLine(b []byte) (int64, error) {
	v, err := strconv.ParseInt(string(bytes.Fields(b)[0]), 10, 64)
	return v * 1024, err
}

func (m *MemInfo) parseMeminfoLine(name, rest []byte) error {
	var err error
	switch string(name) {
	case "MemTotal":
		m.Total, err = parseMeminfoLine(rest)
	case "MemFree":
		m.Free, err = parseMeminfoLine(rest)
	case "MemAvailable":
		m.available, err = parseMeminfoLine(rest)
	case "Buffers":
		m.buffers, err = parseMeminfoLine(rest)
	case "Cached":
		m.cached, err = parseMeminfoLine(rest)
	}
	return err
}
