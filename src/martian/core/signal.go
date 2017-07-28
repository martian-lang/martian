//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian signal handler.
//
package core

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
	mutex   *sync.Mutex
	block   chan int
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
	self := &SignalHandler{}
	self.mutex = &sync.Mutex{}
	self.block = make(chan int)
	self.objects = map[HandlerObject]bool{}
	return self
}

func SetupSignalHandlers() {
	// Handle CTRL-C and kill.
	sigchan := make(chan os.Signal, 4)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, syscall.SIGHUP)
	signal.Notify(sigchan, syscall.SIGTERM)
	signal.Notify(sigchan, syscall.SIGUSR1)
	signal.Notify(sigchan, syscall.SIGUSR2)

	signalHandler = newSignalHandler()
	go func() {
		sig := <-sigchan
		Println("Caught signal %v", sig)

		// Set exit flag
		signalHandler.mutex.Lock()
		signalHandler.exit = true

		// Make sure all goroutines have left critical sections
		for signalHandler.count > 0 {
			signalHandler.mutex.Unlock()
			time.Sleep(1)
			signalHandler.mutex.Lock()
		}
		for object := range signalHandler.objects {
			object.HandleSignal(sig)
		}
		os.Exit(1)
	}()
}
