// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

// Keeps track of the number of running jobs.

package core

import (
	"sync"
)

// A semaphore limiting the number of unique jobs which are active at a time.
type MaxJobsSemaphore struct {
	running map[*Metadata]struct{}
	cond    *sync.Cond
	lock    sync.Mutex
	Limit   int
}

func NewMaxJobsSemaphore(limit int) *MaxJobsSemaphore {
	if limit < 1 {
		panic("Invalid max jobs limit")
	}
	self := &MaxJobsSemaphore{
		running: make(map[*Metadata]struct{}),
		Limit:   limit,
	}
	self.cond = sync.NewCond(&self.lock)
	return self
}

// Wait for this semaphore to have capacity to run this metadata
// object.
//
// If the object is not in the queued or waiting states, it was canceled
// between when the job was enqueued and now.
//
// If the object was already in the semaphore, as may be the case in the
// event of automatic restart if the failure was missed for whatever reason,
// then we only treat the metadata object as having one job running ever.
//
// If nonblocking is true, then if the semaphore cannot be acquired immediately
// (mostly) then it will return false.
func (self *MaxJobsSemaphore) Acquire(metadata *Metadata, nonblocking bool) bool {
	if metadata == nil {
		return false
	}
	if st, ok := metadata.getState(); ok && st != Queued && st != Waiting {
		return false
	}
	// In case this particular metadata object is waiting more than once,
	// make sure to always signal the condition variable once this one
	// successfully acquires.
	defer self.cond.Signal()
	self.lock.Lock()
	defer self.lock.Unlock()
	for len(self.running) >= self.Limit {
		if self.Limit <= 0 {
			return false
		}
		if st, ok := metadata.getState(); ok && st != Queued && st != Waiting {
			return false
		}
		if _, ok := self.running[metadata]; ok {
			return true
		}
		if nonblocking {
			return false
		}
		self.cond.Wait()
	}
	if st, ok := metadata.getState(); ok && st != Queued && st != Waiting {
		return false
	}
	self.running[metadata] = struct{}{}
	return true
}

// Clear this semaphore and release all pending acquisitions.
//
// The semaphore can no longer be used after being cleared this way.
func (self *MaxJobsSemaphore) Clear() {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.Limit = 0
	self.cond.Broadcast()
}

// Check that each metadata object which holds the semaphore is still
// actually running.
func (self *MaxJobsSemaphore) FindDone() {
	self.lock.Lock()
	defer self.lock.Unlock()

	finished := make([]*Metadata, 0, len(self.running))
	for m := range self.running {
		if st, ok := m.getState(); ok && st != Running && st != Queued {
			finished = append(finished, m)
		}
	}
	if len(finished) > 0 {
		// Some metadatas which were believed to be running were not,
		// in fact, running.  Remove them from the semaphore.
		for _, m := range finished {
			delete(self.running, m)
		}
		// If there is now more than one free capacity in the semaphore,
		// notify other waiters.
		spare := self.Limit - len(self.running)
		if spare > 1 {
			self.cond.Broadcast()
		} else if spare == 1 {
			self.cond.Signal()
		}
	}
}

func (self *MaxJobsSemaphore) Release(metadata *Metadata) {
	if metadata == nil {
		return
	}
	self.lock.Lock()
	defer self.lock.Unlock()
	if _, ok := self.running[metadata]; ok {
		delete(self.running, metadata)
		self.cond.Signal()
	}
}

func (self *MaxJobsSemaphore) Current() int {
	self.lock.Lock()
	defer self.lock.Unlock()
	return len(self.running)
}
