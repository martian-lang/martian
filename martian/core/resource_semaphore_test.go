// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package core

import (
	"runtime"
	"testing"
	"time"
)

func TestResourceSemaphoreSetActual(t *testing.T) {
	done := make(chan int)
	go func() {
		sem := NewResourceSemaphore(100, DefaultResourceFormatter("test"))
		if sem.InUse() != 0 {
			t.Errorf("Expected nothing in use, got %d", sem.InUse())
		}
		if diff := sem.UpdateActual(200); diff != 100 {
			t.Errorf("Expected 100 diff, got %d", diff)
		}
		if sem.InUse() != 0 {
			t.Errorf("Expected nothing in use, got %d", sem.InUse())
		}
		if diff := sem.UpdateActual(90); diff != -10 {
			t.Errorf("Expected -10 diff, got %d", diff)
		}
		if sem.InUse() != 10 {
			t.Errorf("Expected 10 in use, got %d", sem.InUse())
		}
		if sem.Reserved() != 0 {
			t.Errorf("Expected none in use, got %d", sem.InUse())
		}
		if err := sem.Acquire(10); err != nil {
			t.Error(err)
		}
		if sem.Available() != 80 {
			t.Errorf("Expected 80 available, got %d", sem.Available())
		}
		if diff := sem.UpdateActual(90); diff != 0 {
			t.Errorf("Expected no diff, got %d", sem.InUse())
		}
		if sem.Available() != 90 {
			t.Errorf("Expected 90 available, got %d", sem.Available())
		}
		if sem.CurrentSize() != 100 {
			t.Errorf("Expected 100 size, got %d", sem.CurrentSize())
		}
		sem.Release(10)
		if err := sem.Acquire(5); err != nil {
			t.Error(err)
		}
		if sem.InUse() != 5 {
			t.Errorf("Expected 5 in use, got %d", sem.InUse())
		}
		if err := sem.Acquire(10); err != nil {
			t.Error(err)
		}
		if sem.InUse() != 15 {
			t.Errorf("Expected 15 in use, got %d", sem.InUse())
		}
		if diff := sem.UpdateActual(85); diff != 0 {
			t.Errorf("Expected no diff, got %d", diff)
		}
		done <- 1
	}()
	timer := time.NewTimer(time.Second * 10)
	select {
	case <-done:
		return
	case <-timer.C:
		t.Errorf("Timed out.")
	}
}

func TestResourceSemaphoreAcquireError(t *testing.T) {
	sem := NewResourceSemaphore(90, DefaultResourceFormatter("test"))
	done := make(chan int)
	go func() {
		if err := sem.Acquire(100); err == nil {
			t.Errorf("Unexpected success.")
		}
		done <- 1
	}()
	timer := time.NewTimer(time.Second * 10)
	select {
	case <-done:
		return
	case <-timer.C:
		t.Errorf("Timed out.")
	}
}

func TestResourceSemaphoreRelease(t *testing.T) {
	done := make(chan int)
	go func() {
		sem := NewResourceSemaphore(40, DefaultResourceFormatter("test"))
		started := make(chan int)
		acquired := make(chan int)
		released := make(chan int)
		allowRelease := make(chan int)
		hasAcquired := make([]bool, 4)
		acquire := func(id int, amount int64, releaseAny bool) {
			started <- id
			if err := sem.Acquire(amount); err != nil {
				t.Error(err)
			}
			hasAcquired[id] = true
			acquired <- id
			if msg := <-allowRelease; msg != id && !releaseAny {
				t.Errorf("Expected allow release %d, got %d", id, msg)
			}
			sem.Release(amount)
			released <- id
		}
		go acquire(0, 30, false)
		if id := <-started; id != 0 {
			t.Errorf("Unexpected id %d", id)
		}
		if msg := <-acquired; msg != 0 {
			t.Errorf("Expected acquire 0, got %d", msg)
		}
		if !hasAcquired[0] {
			t.Errorf("Expected %d to have acquired.", 0)
		}
		if sem.InUse() != 30 {
			t.Errorf("Expected 30 in use, got %d", sem.InUse())
		}
		go acquire(1, 20, true)
		go acquire(2, 20, true)
		if id := <-started; id != 1 && id != 2 {
			t.Errorf("Unexpected id %d", id)
		}
		if id := <-started; id != 1 && id != 2 {
			t.Errorf("Unexpected id %d", id)
		}
		go acquire(3, 35, false)
		// Make sure 1 and 2 get in the queue before 3.
		for sem.QueueLength() != 2 {
			time.Sleep(time.Millisecond)
		}
		if id := <-started; id != 3 {
			t.Errorf("Unexpected id %d", id)
		}
		if hasAcquired[1] {
			t.Errorf("Expected %d to block.", 1)
		}
		if hasAcquired[2] {
			t.Errorf("Expected %d to block.", 2)
		}
		if sem.InUse() != 30 {
			t.Errorf("Expected 30 in use, got %d", sem.InUse())
		}
		allowRelease <- 0
		if msg := <-released; msg != 0 {
			t.Errorf("Expected release 0, got %d", msg)
		}
		if msg := <-acquired; msg != 1 && msg != 2 {
			t.Errorf("Expected acquire 1 or 2, got %d", msg)
		}
		if msg := <-acquired; msg != 1 && msg != 2 {
			t.Errorf("Expected acquire 1 or 2, got %d", msg)
		}
		if !hasAcquired[1] {
			t.Errorf("Expected %d to have acquired.", 1)
		}
		if !hasAcquired[2] {
			t.Errorf("Expected %d to have acquired.", 2)
		}
		if hasAcquired[3] {
			t.Errorf("Expected %d to not have acquired.", 3)
		}
		if sem.InUse() != 40 {
			t.Errorf("Expected 40 in use, got %d", sem.InUse())
		}
		allowRelease <- 1
		if msg := <-released; msg != 1 && msg != 2 {
			t.Errorf("Expected release 1 or 2, got %d", msg)
		}
		if sem.InUse() != 20 {
			t.Errorf("Expected 40 in use, got %d", sem.InUse())
		}
		if hasAcquired[3] {
			t.Errorf("Expected %d to not have acquired.", 3)
		}
		allowRelease <- 2
		if msg := <-released; msg != 1 && msg != 2 {
			t.Errorf("Expected release 1 or 2, got %d", msg)
		}
		if msg := <-acquired; msg != 3 {
			t.Errorf("Expected acquire 3, got %d", msg)
		}
		if sem.InUse() != 35 {
			t.Errorf("Expected 35 in use, got %d", sem.InUse())
		}
		if !hasAcquired[3] {
			t.Errorf("Expected %d to have acquired.", 3)
		}
		allowRelease <- 3
		done <- 1
	}()
	timer := time.NewTimer(time.Second * 25)
	select {
	case <-done:
		return
	case <-timer.C:
		buf := make([]byte, 4096)
		buf = buf[:runtime.Stack(buf, true)]
		t.Log(string(buf))
		t.Errorf("Timed out.")
	}
}

func TestResourceSemaphoreActualRelease(t *testing.T) {
	done := make(chan int)
	go func() {
		sem := NewResourceSemaphore(100, DefaultResourceFormatter("test"))
		if sem.InUse() != 0 {
			t.Errorf("Expected nothing in use, got %d", sem.InUse())
		}
		if diff := sem.UpdateActual(20); diff != -80 {
			t.Errorf("Expected -80 diff, got %d", diff)
		}
		if sem.InUse() != 80 {
			t.Errorf("Expected 80 in use, got %d", sem.InUse())
		}
		ch := make(chan int)
		acq := false
		go func() {
			if err := sem.Acquire(25); err != nil {
				t.Error(err)
			}
			acq = true
			ch <- 1
			ch <- 1
			sem.Release(25)
			ch <- 1
		}()
		if sem.InUse() != 80 {
			t.Errorf("Expected 80 in use, got %d", sem.InUse())
		}
		if acq {
			t.Errorf("Should not have been able to acquire yet.")
		}
		if diff := sem.UpdateActual(25); diff != -75 {
			t.Errorf("Expected -75 diff, got %d", diff)
		}
		<-ch
		if !acq {
			t.Errorf("Should have acquired.")
		}
		if sem.InUse() != 100 {
			t.Errorf("Expected 100 in use, got %d", sem.InUse())
		}
		<-ch
		<-ch
		if sem.InUse() != 75 {
			t.Errorf("Expected 75 in use, got %d", sem.InUse())
		}
		done <- 1
	}()
	timer := time.NewTimer(time.Second * 10)
	select {
	case <-done:
		return
	case <-timer.C:
		t.Errorf("Timed out.")
	}
}
