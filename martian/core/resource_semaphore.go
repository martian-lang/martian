// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package core

// A semaphore for reserving resources against a starting value, as well as
// instantaneous queries of the actual availability (in case reservations
// get exceeded).

import (
	"fmt"
	"github.com/martian-lang/martian/martian/util"
	"sync"
)

type waiter struct {
	amount int64
	ready  chan<- struct{} // Closed when semaphore acquired.
}

// A semaphore type which allows for the maxium size of things entering the
// semaphore to be dynamically reduced based on observed resource availability.
type ResourceSemaphore struct {
	Name string

	// The maximum that's allowed to be reserved, ever.
	maxSize int64

	// The maximum that can be reserved right now, given the last seen
	// actual resouce availability.
	curSize int64

	// The amount currently reserved.  This amount can exceed curSize but not
	// maxSize.
	reserved int64
	mu       sync.Mutex
	waiters  []waiter
}

// Create a new semaphore with the given capactiy.
func NewResourceSemaphore(size int64, name string) *ResourceSemaphore {
	return &ResourceSemaphore{
		Name:    name,
		maxSize: size,
		curSize: size,
	}
}

// Reserve n of the resource.  Block until it is available.  Returns an error
// if more was requested than is possible to serve.
func (self *ResourceSemaphore) Acquire(n int64) error {
	self.mu.Lock()
	if self.curSize-self.reserved >= n && len(self.waiters) == 0 {
		// return immediately.
		self.reserved += n
		self.mu.Unlock()
		return nil
	}

	if n > self.maxSize {
		// This can never be served.
		self.mu.Unlock()
		return fmt.Errorf("Tried to aquire %d %s, when the maximum is %d.",
			n, self.Name, self.maxSize)
	}

	if len(self.waiters) == 0 && self.curSize-self.reserved > 0 {
		util.LogInfo("jobmngr", "Need %d %s to start the next job (%d available).  Waiting for jobs to complete.",
			n, self.Name, self.curSize-self.reserved)
	}

	// Enqueue.
	ready := make(chan struct{})
	w := waiter{amount: n, ready: ready}
	self.waiters = append(self.waiters, w)
	self.mu.Unlock()

	<-ready
	return nil
}

// Release n of the resource.
func (self *ResourceSemaphore) Release(n int64) {
	self.mu.Lock()
	self.reserved -= n
	if self.reserved < 0 {
		self.mu.Unlock()
		panic("semaphore: bad release")
	}
	self.runJobs()
	self.mu.Unlock()
}

func (self *ResourceSemaphore) runJobs() {
	for i, waiter := range self.waiters {
		if self.curSize-self.reserved < waiter.amount {
			if self.curSize-self.reserved > 0 {
				util.LogInfo("jobmngr", "Need %d %s to start the next job (%d available).  Waiting for jobs to complete.",
					waiter.amount, self.Name, self.curSize-self.reserved)
			}
			self.waiters = self.waiters[i:]
			return
		}
		self.reserved += waiter.amount
		close(waiter.ready)
	}
	self.waiters = nil
}

// Get the current amount of resources in use.  This includes both reserved
// resources and resources for which their usage is unaccounted for.
func (self *ResourceSemaphore) InUse() int64 {
	self.mu.Lock()
	used := self.maxSize - self.curSize + self.reserved
	self.mu.Unlock()
	return used
}

// Get the current amount of explicitly reserved resources.
func (self *ResourceSemaphore) Reserved() int64 {
	self.mu.Lock()
	res := self.reserved
	self.mu.Unlock()
	return res
}

// Get the current amount of available resources.
func (self *ResourceSemaphore) Available() int64 {
	self.mu.Lock()
	res := self.curSize - self.reserved
	self.mu.Unlock()
	return res
}

// Get the current amount of resources which are reservable (including those
// already reserved).
func (self *ResourceSemaphore) CurrentSize() int64 {
	self.mu.Lock()
	res := self.curSize
	self.mu.Unlock()
	return res
}

// Get the number of items waiting on the semaphore.
func (self *ResourceSemaphore) QueueLength() int {
	self.mu.Lock()
	length := len(self.waiters)
	self.mu.Unlock()
	return length
}

// Set the current actual availability, e.g. by checking free memory.
// Returns the difference between the unreserved and actual.  A negative return
// value indicates either a job which is using more memory than it reserved,
// or some other process on the system is using memory.
func (self *ResourceSemaphore) UpdateActual(n int64) int64 {
	self.mu.Lock()
	actualSize := n + self.reserved
	oldSize := self.curSize
	if actualSize > self.maxSize {
		self.curSize = self.maxSize
	} else {
		self.curSize = actualSize
	}
	// There may have been jobs blocked on the actual availability.
	if oldSize < self.curSize {
		self.runJobs()
	}
	self.mu.Unlock()
	return actualSize - self.maxSize
}

// Change the current semaphore size.  This is is for cases where the resource
// limit may change but the current consumption is invisible.  It is logically
// equivalent to self.UpdateActual(n, self.Reserved()), though without the
// potential race conditions.
func (self *ResourceSemaphore) UpdateSize(n int64) {
	self.mu.Lock()
	defer self.mu.Unlock()
	oldSize := self.curSize
	self.curSize = n
	if oldSize < self.curSize {
		self.runJobs()
	}
}

// Set the current actual availability based on the current free amount and the
// amount of the reserved usage which is actually in use.  This handles the
// case where, for example, 30 of 32 GB of memory are reserved, but only 16GB
// has actually been committed so far, so 16GB appears to be free.
func (self *ResourceSemaphore) UpdateFreeUsed(free, usedReservation int64) int64 {
	actualSize := free + usedReservation
	self.mu.Lock()
	oldSize := self.curSize
	if usedReservation <= self.reserved {
		if actualSize > self.maxSize {
			self.curSize = self.maxSize
		} else {
			self.curSize = actualSize
		}
	} else {
		// We're using more than we think we're using.  Adjust
		// the usage cap appropriately.
		adjust := usedReservation - self.reserved
		if actualSize > self.maxSize-adjust {
			self.curSize = self.maxSize - adjust
		} else {
			self.curSize = actualSize - adjust
		}
	}
	// There may have been jobs blocked on the actual availability.
	if oldSize < self.curSize {
		self.runJobs()
	}
	self.mu.Unlock()
	return actualSize - self.maxSize
}
