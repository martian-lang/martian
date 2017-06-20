//
// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
//
// A semaphore for reserving resources against a starting value, as well as
// instantaneous queries of the actual availability (in case reservations
// get exceeded).
//
package core

import (
	"fmt"
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

	if len(self.waiters) == 0 {
		LogInfo("jobmngr", "Attempted to reserve %d %s, but only %d were available.",
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
			LogInfo("jobmngr", "Attempted to reserve %d %s, but only %d were available.",
				waiter.amount, self.Name, self.curSize-self.reserved)
			self.waiters = self.waiters[i:]
			return
		}
		self.reserved += waiter.amount
		close(waiter.ready)
	}
	self.waiters = nil
}

// Get the current amount of resources in use
func (self *ResourceSemaphore) InUse() int64 {
	self.mu.Lock()
	used := self.maxSize - self.curSize + self.reserved
	self.mu.Unlock()
	return used
}

// Set the current actual availability, e.g. by checking free memory.
// Returns the difference between the unreserved and actual.  A negative return
// value indicates either a job which is using more memory than it reserved,
// or some other process on the system is using memory.
func (self *ResourceSemaphore) UpdateActual(n int64) int64 {
	self.mu.Lock()
	// TODO: it would be better to use the total resource usage of owned
	// martian jobs here instead of the reservation.  That can be tricky to
	// compute, however.
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
