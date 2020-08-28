// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

package core

/*
Data structures for managing stage I/O performance information.
On linux, this information is pulled from /proc/[pid]/io.

From the /proc manual page:

       /proc/[pid]/io (since kernel 2.6.20)
              This file contains I/O statistics for the process, for example:

                  # cat /proc/3828/io
                  rchar: 323934931
                  wchar: 323929600
                  syscr: 632687
                  syscw: 632675
                  read_bytes: 0
                  write_bytes: 323932160
                  cancelled_write_bytes: 0

              The fields are as follows:

              rchar: characters read
                     The number of bytes which this task has caused to be read
                     from storage.  This is simply the sum of bytes which this
                     process passed to read(2) and similar system  calls.   It
                     includes things such as terminal I/O and is unaffected by
                     whether or not actual physical disk I/O was required (the
                     read might have been satisfied from pagecache).

              wchar: characters written
                     The  number of bytes which this task has caused, or shall
                     cause to be written to disk.  Similar caveats apply  here
                     as with rchar.

              syscr: read syscalls
                     Attempt  to  count the number of read I/O operations—that
                     is, system calls such as read(2) and pread(2).

              syscw: write syscalls
                     Attempt to count the number of write I/O  operations—that
                     is, system calls such as write(2) and pwrite(2).

              read_bytes: bytes read
                     Attempt  to  count the number of bytes which this process
                     really did cause to be fetched from  the  storage  layer.
                     This is accurate for block-backed filesystems.

              write_bytes: bytes written
                     Attempt  to  count the number of bytes which this process
                     caused to be sent to the storage layer.

              cancelled_write_bytes:
                     The big inaccuracy here is truncate.  If a process writes
                     1MB  to a file and then deletes the file, it will in fact
                     perform no writeout.  But it will have been accounted  as
                     having  caused  1MB of write.  In other words: this field
                     represents the number of bytes which this process  caused
                     to not happen, by truncating pagecache.  A task can cause
                     "negative" I/O too.  If this task  truncates  some  dirty
                     pagecache, some I/O which another task has been accounted
                     for (in its write_bytes) will not be happening.
*/

import (
	"math"
	"time"
)

// Collects statistics based on observed process tree IO usage.
type IoStats struct {
	Total   IoAmount `json:"total,omitempty"`
	RateMax IoRate   `json:"max,omitempty"`
	RateDev IoRate   `json:"dev,omitempty"`
}

type IoStatsBuilder struct {
	IoStats

	// The per-pid usage seen last time the tree was scanned.
	lastPids map[int]*IoAmount
	// The amount used by PIDs which were seen in the past but
	// not in a subsequent scan of the process tree.
	deadUsage IoAmount

	weightedSumSquared IoRate
	lastMeasurement    time.Time
	start              time.Time
}

func NewIoStatsBuilder() *IoStatsBuilder {
	t := time.Now()
	return &IoStatsBuilder{
		lastPids:        make(map[int]*IoAmount),
		lastMeasurement: t,
		start:           t,
	}
}

// Update the stats object with the current per-pid IO amounts.
func (self *IoStatsBuilder) Update(current map[int]*IoAmount, now time.Time) {
	seconds := now.Sub(self.lastMeasurement).Seconds()
	var total IoAmount
	for pid, amt := range current {
		total.Increment(amt)
		delete(self.lastPids, pid)
	}
	for _, amt := range self.lastPids {
		self.deadUsage.Increment(amt)
	}
	total.Increment(&self.deadUsage)
	self.lastPids = current
	self.lastMeasurement = now
	diff := self.Total.update(&total)
	if seconds > 0 {
		rate := diff.rate(seconds)
		self.RateMax.TakeMax(rate)
		rate.weightSquared(seconds)
		self.weightedSumSquared.Increment(rate)
		if t := now.Sub(self.start).Seconds(); t > 0 {
			self.RateDev = self.weightedSumSquared.computeStdDev(
				&self.Total, t)
		}
	}
}

// Collects a total number of read/write IO operations.
type IoAmount struct {
	Read  IoValues `json:"read"`
	Write IoValues `json:"write"`
}

// Increment this value by another.
func (self *IoAmount) Increment(other *IoAmount) {
	self.Read.Increment(other.Read)
	self.Write.Increment(other.Write)
}

func (self *IoAmount) update(other *IoAmount) (diff IoAmount) {
	self.Read.update(other.Read, &diff.Read)
	self.Write.update(other.Write, &diff.Write)
	return diff
}

func (self *IoAmount) rate(seconds float64) *IoRate {
	if seconds <= 0 {
		return new(IoRate)
	}
	return &IoRate{
		Read:  self.Read.rate(seconds),
		Write: self.Write.rate(seconds),
	}
}

// Stores cumulative values for IO metrics.
type IoValues struct {
	// The number of io syscalls issued.  These may have been against non-block
	// devices, such as sockets, terminals or a psudo-filesystem such as /proc.
	// For block IO operations, see rusage.
	Syscalls int64 `json:"sysc"`

	// The number of bytes transferred to or from a block device.  Even when a
	// read or write is made against a block device, it may not be counted in
	// this number if it was served from cache, or if the file was truncated
	// or unlinked before it was synced to disk.
	BlockBytes int64 `json:"bytes"`
}

func (self *IoValues) update(other IoValues, diff *IoValues) {
	diff.Syscalls += (other.Syscalls - self.Syscalls)
	self.Syscalls = other.Syscalls
	diff.BlockBytes += (other.BlockBytes - self.BlockBytes)
	self.BlockBytes = other.BlockBytes
}

func (self *IoValues) rate(seconds float64) IoRateValues {
	if seconds <= 0 {
		return IoRateValues{}
	}
	return IoRateValues{
		Syscalls:   float64(self.Syscalls) / seconds,
		BlockBytes: float64(self.BlockBytes) / seconds,
	}
}

// Increment this value by another.
func (self *IoValues) Increment(other IoValues) {
	self.Syscalls += other.Syscalls
	self.BlockBytes += other.BlockBytes
}

// Represents a rate of change for IoAmount.
type IoRate struct {
	Read  IoRateValues `json:"read"`
	Write IoRateValues `json:"write"`
}

// Increment this rate by another.
func (self *IoRate) Increment(other *IoRate) {
	self.Read.Increment(other.Read)
	self.Write.Increment(other.Write)
}

// Update this rate to be the maximum of itself and another.
func (self *IoRate) TakeMax(other *IoRate) {
	self.Read.TakeMax(other.Read)
	self.Write.TakeMax(other.Write)
}

// Update x_i = δt * x_i^2.
func (self *IoRate) weightSquared(seconds float64) {
	self.Read.weightSquared(seconds)
	self.Write.weightSquared(seconds)
}

// Compute the standard deviation of rate, given this object as the weighted
// sum squared sum_i[δt_i*x_i^2].
func (sumSq *IoRate) computeStdDev(total *IoAmount, seconds float64) IoRate {
	return IoRate{
		Read:  sumSq.Read.computeStdDev(total.Read, seconds),
		Write: sumSq.Write.computeStdDev(total.Write, seconds),
	}
}

// IoValues per second.
type IoRateValues struct {
	// The rate at which IO syscalls were issued.  These may have been against
	// non-block devices, such as sockets, terminals or a psudo-filesystem
	// such as /proc.
	Syscalls float64 `json:"sysc"`

	// The rate at which bytes were transferred to or from a block device.
	// Even when a read or write is made against a block device, it may not be
	// counted in this number if it was served from cache, or if the file was
	// truncated or unlinked before it was synced to disk.
	BlockBytes float64 `json:"bytes"`
}

// Increment this rate by another.
func (self *IoRateValues) Increment(other IoRateValues) {
	self.Syscalls += other.Syscalls
	self.BlockBytes += other.BlockBytes
}

// Update this rate to be the maximum of itself and another.
func (self *IoRateValues) TakeMax(other IoRateValues) {
	if self.Syscalls < other.Syscalls {
		self.Syscalls = other.Syscalls
	}
	if self.BlockBytes < other.BlockBytes {
		self.BlockBytes = other.BlockBytes
	}
}

// Update x_i = δt * x_i^2.
func (self *IoRateValues) weightSquared(seconds float64) {
	self.Syscalls = self.Syscalls * self.Syscalls * seconds
	self.BlockBytes = self.BlockBytes * self.BlockBytes * seconds
}

func (self IoRateValues) squared() IoRateValues {
	return IoRateValues{
		Syscalls:   self.Syscalls * self.Syscalls,
		BlockBytes: self.BlockBytes * self.BlockBytes,
	}
}

func (self IoRateValues) sub(other IoRateValues) IoRateValues {
	return IoRateValues{
		Syscalls:   self.Syscalls - other.Syscalls,
		BlockBytes: self.BlockBytes - other.BlockBytes,
	}
}

func (self IoRateValues) sqrt() IoRateValues {
	if self.Syscalls < 0 {
		self.Syscalls = 0
	}
	if self.BlockBytes < 0 {
		self.BlockBytes = 0
	}
	return IoRateValues{
		Syscalls:   math.Sqrt(self.Syscalls),
		BlockBytes: math.Sqrt(self.BlockBytes),
	}
}

// Compute the standard deviation of rate, given this object as the weighted
// sum squared sum_i[δt_i*x_i^2]
//
//   stdDev = sqrt(var)
//   x = total amount
//   t = total time
//   x_i = δx_i / δt_i
//   w_i = δt_i
//   mean = m = sum_i[ δt_i * x_i ] / sum_i[δt_i] = sum_i[δx_i] / t
//        = x / t
//   var = sum_i[δt_i*(x_i-mx)^2] / sum_i[δt_i]
//       = (sum_i[δt_i*x_i^2] +
//          sum_i[δt_i*mx^2] -
//          sum_i[δt_i*x_i*mx]) / t
//       = sum_i[δt_i*x_i^2] / t - mx^2
func (sumSq *IoRateValues) computeStdDev(total IoValues, seconds float64) IoRateValues {
	if seconds <= 0 {
		return IoRateValues{}
	}
	return IoRateValues{
		Syscalls:   sumSq.Syscalls / seconds,
		BlockBytes: sumSq.BlockBytes / seconds,
	}.sub(total.rate(seconds).squared()).sqrt()
}
