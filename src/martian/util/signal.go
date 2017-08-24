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
	"time"
)

type HandlerObject interface {
	HandleSignal(sig os.Signal)
}

type SignalHandler struct {
	count   int
	exit    bool
	mutex   sync.Mutex
	block   chan int
	sigchan chan os.Signal
	objects map[HandlerObject]bool
}

var signalHandler *SignalHandler = nil

func EnterCriticalSection() {
	signalHandler.mutex.Lock()
	if signalHandler.exit {
		// Block other goroutines from entering critical section if exit flag has been set
		signalHandler.mutex.Unlock()
		<-signalHandler.block
	}
	signalHandler.count += 1
	signalHandler.mutex.Unlock()
}

func ExitCriticalSection() {
	signalHandler.mutex.Lock()
	signalHandler.count -= 1
	signalHandler.mutex.Unlock()
}

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
		sigchan: make(chan os.Signal, 6),
	}
}

// Notify this handler of signals.
func (self *SignalHandler) Notify() {
	signal.Notify(self.sigchan, os.Interrupt)
	if !SignalIsIgnored(syscall.SIGHUP) {
		signal.Notify(self.sigchan, syscall.SIGHUP)
	}
	signal.Notify(self.sigchan, syscall.SIGTERM)
	signal.Notify(self.sigchan, syscall.SIGUSR1)
	signal.Notify(self.sigchan, syscall.SIGUSR2)
}

// Kill this process cleanly.
func Suicide() {
	Println("Shutting down.")
	if signalHandler == nil {
		os.Exit(1)
	}
	signalHandler.sigchan <- syscall.Signal(-1)
}

func SetupSignalHandlers() {
	signalHandler = newSignalHandler()
	signalHandler.Notify()
	sigchan := signalHandler.sigchan

	go func() {
		sig := <-sigchan
		if sig != syscall.Signal(-1) {
			Println("Caught signal %v", sig)
		}

		// Set exit flag
		signalHandler.mutex.Lock()
		signalHandler.exit = true

		// Make sure all goroutines have left critical sections
		for signalHandler.count > 0 {
			signalHandler.mutex.Unlock()
			time.Sleep(time.Millisecond)
			signalHandler.mutex.Lock()
		}
		for object := range signalHandler.objects {
			object.HandleSignal(sig)
		}
		os.Exit(1)
	}()
}
