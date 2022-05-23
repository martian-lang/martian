//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//
// Martian signal handler.
//

package util

import (
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

// Interface for objects which need to perform cleanup operations when the
// process is being terminated by a handled signal.
//
// The HandleSignal method is called after all critical sections have
// completed.  Attempting to enter a critical section from within the
// HandleSignal method will result in a deadlock.
type HandlerObject interface {
	HandleSignal(sig os.Signal)
}

type sigHandler struct {
	objects         map[HandlerObject]struct{}
	sigchan         chan os.Signal
	criticalSection sync.RWMutex
	mutex           sync.Mutex
}

var signalHandler sigHandler

// EnterCriticalSection should be called before performing a sequence of
// operations which should be logically atomic with respect to signals.
// Handled signals will not terminate the process until ExitCriticalSection is
// called.  If the process has already been signaled, EnterCriticalSection
// will block.
//
// Note that EnterCriticalSection should never be called when already in
// a critical section, as it may deadlock if the process is signaled between
// the first and second call.
func EnterCriticalSection() {
	signalHandler.criticalSection.RLock()
}

// ExitCriticalSection should be called after EnterCriticalSection once
// the logically atomic sequence of operations has completed.
func ExitCriticalSection() {
	signalHandler.criticalSection.RUnlock()
}

// Registers an object as having cleanup work which should be allowed to
// complete before the process terminates due to a handled signal.
func RegisterSignalHandler(object HandlerObject) {
	signalHandler.mutex.Lock()
	if signalHandler.objects == nil {
		signalHandler.objects = make(map[HandlerObject]struct{})
	}
	signalHandler.objects[object] = struct{}{}
	signalHandler.mutex.Unlock()
}

func UnregisterSignalHandler(object HandlerObject) {
	signalHandler.mutex.Lock()
	delete(signalHandler.objects, object)
	signalHandler.mutex.Unlock()
}

// Notify this handler of signals.
func (self *sigHandler) notify() {
	if self.sigchan == nil {
		self.sigchan = make(chan os.Signal, len(HANDLED_SIGNALS)+1)
	}
	for _, sig := range HANDLED_SIGNALS {
		if sig != syscall.SIGHUP || !signal.Ignored(syscall.SIGHUP) {
			signal.Notify(self.sigchan, sig)
		}
	}
}

// Kill this process cleanly, after waiting for critical sections
// and handlers to complete.  Note that this method may return.
func Suicide(success bool) {
	Println("%s Shutting down.", Timestamp())
	if signalHandler.sigchan == nil {
		if success {
			os.Exit(0)
		}
		os.Exit(1)
	}
	if success {
		signalHandler.sigchan <- syscall.Signal(-1)
	} else {
		signalHandler.sigchan <- syscall.Signal(-2)
	}
	// We don't want to exit immediately, since handlers may still need to
	// run, but we also don't want to return, because that would be
	// surprising.
	runtime.Goexit()
}

// Set up a signal handler object to support testing of code which
// requires it, without actually registering for signal notifications.
//
// Deprecated: No longer required.
func MockSignalHandlersForTest() {}

// Initializes the global signal handler.
func SetupSignalHandlers() {
	signalHandler.notify()
	sigchan := signalHandler.sigchan

	go func() {
		sig := <-sigchan
		if sig != syscall.Signal(-1) && sig != syscall.Signal(-2) {
			Println("%s Caught signal %v", Timestamp(), sig)
		}

		signalHandler.criticalSection.Lock()
		signalHandler.mutex.Lock()

		var wg sync.WaitGroup
		for object := range signalHandler.objects {
			wg.Add(1)
			go func(wg *sync.WaitGroup, object HandlerObject) {
				defer wg.Done()
				object.HandleSignal(sig)
			}(&wg, object)
		}
		wg.Wait()
		if sig == syscall.Signal(-1) {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}()
}
