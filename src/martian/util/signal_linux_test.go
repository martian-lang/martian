// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package util

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestSignalIsIgnored(t *testing.T) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	if SignalIsIgnored(syscall.SIGUSR1) {
		t.Errorf("Signal should not be ignored here.")
	}
	signal.Stop(c)
	signal.Ignore(syscall.SIGUSR1)
	if !SignalIsIgnored(syscall.SIGUSR1) {
		t.Error("Signal should be ignored.")
	}
	signal.Reset(syscall.SIGUSR1)
}
