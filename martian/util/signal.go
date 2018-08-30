//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian signal handler.
//

package util

import (
	"os"
	"os/signal"
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

type SignalHandler struct {
	criticalSection sync.RWMutex
	mutex           sync.Mutex
	block           chan int
	sigchan         chan os.Signal
	objects         map[HandlerObject]bool
}

var signalHandler *SignalHandler = nil

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
	signalHandler.objects[object] = true
	signalHandler.mutex.Unlock()
}

func UnregisterSignalHandler(object HandlerObject) {
	signalHandler.mutex.Lock()
	delete(signalHandler.objects, object)
	signalHandler.mutex.Unlock()
}

func newSignalHandler() *SignalHandler {
	return &SignalHandler{
		block:   make(chan int),
		objects: make(map[HandlerObject]bool),
		sigchan: make(chan os.Signal, len(HANDLED_SIGNALS)+1),
	}
}

// After a call to SetupSignalHandlers, these signals will be handled
// by waiting for all pending critical sections to complete, running
// all registered handlers, and then exiting with return code 1
var HANDLED_SIGNALS = [...]os.Signal{
	os.Interrupt,
	syscall.SIGHUP,
	syscall.SIGTERM,
	syscall.SIGUSR1,
	syscall.SIGUSR2,
}

// Notify this handler of signals.
func (self *SignalHandler) Notify() {
	for _, sig := range HANDLED_SIGNALS {
		if sig != syscall.SIGHUP || !signal.Ignored(syscall.SIGHUP) {
			signal.Notify(self.sigchan, sig)
		}
	}
}

// Kill this process cleanly, after waiting for critical sections
// and handlers to complete.
func Suicide(success bool) {
	Println("%s Shutting down.", Timestamp())
	if signalHandler == nil {
		os.Exit(1)
	}
	if success {
		signalHandler.sigchan <- syscall.Signal(-1)
	} else {
		signalHandler.sigchan <- syscall.Signal(-2)
	}
}

// Initializes the global signal handler.
func SetupSignalHandlers() {
	signalHandler = newSignalHandler()
	signalHandler.Notify()
	sigchan := signalHandler.sigchan

	go func() {
		sig := <-sigchan
		if sig != syscall.Signal(-1) && sig != syscall.Signal(-2) {
			Println("%s Caught signal %v", Timestamp(), sig)
		}

		signalHandler.criticalSection.Lock()
		signalHandler.mutex.Lock()

		for object := range signalHandler.objects {
			object.HandleSignal(sig)
		}
		if sig == syscall.Signal(-1) {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}()
}
