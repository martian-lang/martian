//
// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.
//

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/trace"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/util"
)

const WAIT_SECS = 6

// Remove any buffered items from the channel.
func flushChannel(c <-chan struct{}) {
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}

// Pipestance runner.
func runLoop(pipestanceBox *pipestanceHolder, stepSecs time.Duration,
	vdrMode core.VdrMode, noExit bool, localJobDone <-chan struct{}) {
	pipestanceBox.getPipestance().LoadMetadata(context.Background())

	t := time.NewTimer(0)
	if !t.Stop() {
		<-t.C
	}
	for {
		flushChannel(localJobDone)
		hadProgress := loopBody(pipestanceBox, vdrMode, noExit)

		if !hadProgress {
			// Wait for a either stepSecs or until a local job finishes.
			t.Reset(stepSecs)
			select {
			case <-t.C:
			case <-localJobDone:
				if !t.Stop() {
					<-t.C
				}
			}
			if !pipestanceBox.lastLogCheck.IsZero() &&
				time.Since(pipestanceBox.lastLogCheck) > time.Minute {
				if err := util.VerifyLogFile(); err != nil {
					util.PrintError(err, "runtime",
						"Pipestance directory seems to have disappeared.")
					util.Suicide(false)
				}
			}
			// During the idle portion of the run loop is a good time to
			// run the GC.  We do this after the sleep because StepNodes
			// launches jobs on goroutines, and it's better to give them
			// time to get to the point where they're waiting on the
			// subprocess (or, in cluster mode, possibly finish waiting)
			// before the GC runs.
			runtime.GC()
		}
	}
}

func loopBody(pipestanceBox *pipestanceHolder,
	vdrMode core.VdrMode, noExit bool) bool {
	pipestance := pipestanceBox.getPipestance()
	ctx, task := trace.NewTask(context.Background(), "update")
	defer task.End()
	pipestance.RefreshState(ctx)

	// Check for completion states.
	state := pipestance.GetState(ctx)
	if state == core.Complete || state == core.DisabledState {
		pipestanceBox.UpdateState(state.Prefixed(core.CleanupPrefix))
		cleanupCompleted(pipestance, pipestanceBox, vdrMode, noExit, ctx)
		return false
	} else if state == core.Failed {
		if pipestanceBox.showedFailed {
			pipestanceBox.UpdateState(state)
		} else {
			pipestanceBox.UpdateState(state.Prefixed(core.CleanupPrefix))
		}
		if !attemptRetry(pipestance, pipestanceBox, ctx) {
			pipestance.Unlock()
			cleanupFailed(pipestance, pipestanceBox, noExit, ctx)
		}
		return false
	} else {
		pipestanceBox.UpdateState(state)
		// If we went from failed to something else, allow the failure message to
		// be shown once if we fail again.
		pipestanceBox.showedFailed = false

		// Check job heartbeats.
		pipestance.CheckHeartbeats(ctx)

		// Step all nodes.
		return pipestance.StepNodes(ctx)
	}
}

func attemptRetry(pipestance *core.Pipestance, pipestanceBox *pipestanceHolder,
	outerCtx context.Context) bool {
	ctx, task := trace.NewTask(outerCtx, "attemptRetry")
	defer task.End()

	if pipestanceBox.readOnly {
		return false
	}
	canRetry := false
	var transient_log string
	if pipestanceBox.consumeRetry() {
		canRetry, transient_log = pipestance.IsErrorTransient()
	}
	if transient_log != "" && !pipestanceBox.showedFailed {
		pipestanceBox.UpdateError(transient_log)
	}
	if canRetry {
		pipestanceBox.UpdateState(core.Failed.Prefixed(core.RetryPrefix))
		if pipestanceBox.retryWait > 0 {
			util.LogInfo("runtime",
				"Waiting %s before attempting a retry.",
				pipestanceBox.retryWait.String())
			time.Sleep(pipestanceBox.retryWait)
		}
		// Heartbeat failures often come in clusters.  Look for any others
		// which have come in since failure was detected so that all of
		// those failures get batched up into a single retry.
		pipestance.RefreshState(ctx)
		pipestance.CheckHeartbeats(ctx)
		// Check that no non-transient failures happened in the mean time.
		canRetry, transient_log = pipestance.IsErrorTransient()
		if !canRetry {
			if transient_log != "" && !pipestanceBox.showedFailed {
				pipestanceBox.UpdateError(transient_log)
			}
			return false
		}

		pipestance.Unlock()
		if transient_log != "" {
			util.LogInfo("runtime",
				"Transient error detected.  Log content:\n\n%s\n",
				transient_log)
		}
		util.LogInfo("runtime", "Attempting retry.")
		if err := pipestanceBox.restart(ctx); err != nil {
			util.LogInfo("runtime", "Retry failed:\n%v\n", err)
			// Let the next loop around actually handle the failure.
		}
	}
	return canRetry
}

func cleanupCompleted(pipestance *core.Pipestance, pipestanceBox *pipestanceHolder,
	vdrMode core.VdrMode, noExit bool, ctx context.Context) {
	r := trace.StartRegion(ctx, "cleanupCompleted")
	defer r.End()
	if pipestanceBox.readOnly {
		pipestanceBox.UpdateState(core.Complete)
		util.Println("Pipestance completed successfully, staying alive because --inspect given.\n")
		return
	}
	pipestanceBox.cleanupLock.Lock()
	defer pipestanceBox.cleanupLock.Unlock()
	if vdrMode == core.VdrDisable {
		util.LogInfo("runtime", "VDR disabled. No files killed.")
	} else {
		killReport := pipestance.VDRKill()
		util.LogInfo("runtime", "VDR killed %d files, %s.",
			killReport.Count, humanize.Bytes(killReport.Size))
	}
	trace.WithRegion(ctx, "PostProcess", pipestance.PostProcess)
	pipestance.Unlock()
	pipestance.OnFinishHook(ctx)
	updateComplete := pipestanceBox.UpdateState(core.Complete)
	if noExit {
		util.Println("Pipestance completed successfully, staying alive because --noexit given.\n")
		runtime.GC()
		// Don't return; otherwise we'll repeatedly try to clean up.
		go completedLogCheck()
		runtime.Goexit()
	} else {
		if pipestanceBox.enableUI {
			// Give time for web ui client to get last update.
			util.Println("Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
			time.Sleep(time.Second * time.Duration(WAIT_SECS))
		}
		util.Println("Pipestance completed successfully!\n")
		if updateComplete != nil {
			<-updateComplete
		}
		util.Suicide(true)
	}
}

// Check to see if the pipestance directory was deleted.  If it was, exit.
// This is used when mrp is lanuched with `--noexit` to make sure mrp doesn't
// outlive its usefulness.
func completedLogCheck() {
	for {
		time.Sleep(time.Minute)
		if err := util.VerifyLogFile(); err != nil {
			util.PrintError(err, "runtime",
				"Pipestance directory seems to have disappeared.")
			util.Suicide(true)
		}
	}
}

func cleanupFailed(pipestance *core.Pipestance, pipestanceBox *pipestanceHolder,
	noExit bool, ctx context.Context) {
	r := trace.StartRegion(ctx, "cleanupFailed")
	defer r.End()
	if pipestanceBox.readOnly {
		pipestanceBox.UpdateState(core.Failed)
		if !pipestanceBox.showedFailed {
			pipestanceBox.showedFailed = true
			util.Println("Pipestance failed, staying alive because --inspect given.\n")
		}
		return
	}
	pipestanceBox.cleanupLock.Lock()
	defer pipestanceBox.cleanupLock.Unlock()
	defer func() { pipestanceBox.showedFailed = true }()
	var serverUpdate chan struct{}
	if !pipestanceBox.showedFailed {
		pipestance.OnFinishHook(ctx)
		if _, _, _, log, kind, errPaths := pipestance.GetFatalError(); kind == "assert" {
			// Print preflight check failures.
			util.Println("\n[%s] %s\n", "error", log)
			if log != "" {
				pipestanceBox.UpdateError(log)
			} else {
				pipestanceBox.UpdateError(fmt.Sprintf(
					"Assertion failed.  See logs at:\n%s",
					strings.Join(errPaths, "\n")))
			}
			serverUpdate = pipestanceBox.UpdateState(core.Failed)
			if serverUpdate != nil {
				<-serverUpdate
			}
			util.Suicide(false)
		} else if len(errPaths) > 0 {
			// Build relative path to _errors file
			errPath, _ := filepath.Rel(filepath.Dir(pipestance.GetPath()), errPaths[0])

			if log != "" {
				util.Println(`
[error] Pipestance failed. Error log at:
%s

Log message:
%s
`, errPath, log)
				pipestanceBox.UpdateError(fmt.Sprintf(
					"Pipestance failed. Full log at:\n%s\n%s",
					strings.Join(errPaths, "\n"), log))
			} else {
				// Print path to _errors metadata file in failed stage.
				util.Println(
					"\n[error] Pipestance failed. Please see log at:\n%s\n",
					errPath)
				pipestanceBox.UpdateError(fmt.Sprintf(
					"Pipestance failed. See logs at:\n%s",
					strings.Join(errPaths, "\n")))
			}
			serverUpdate = pipestanceBox.UpdateState(core.Failed)
		}
	} else {
		serverUpdate = pipestanceBox.UpdateState(core.Failed)
	}
	if noExit {
		// If pipestance failed but we're staying alive, only print this once
		// as long as we stay failed.
		if !pipestanceBox.showedFailed {
			util.Println("Pipestance failed, staying alive because --noexit given.\n")
		}
	} else {
		if pipestanceBox.enableUI {
			// Give time for web ui client to get last update.
			util.Println("Waiting %d seconds for UI to do final refresh.", WAIT_SECS)
			time.Sleep(time.Second * time.Duration(WAIT_SECS))
			util.Println("Pipestance failed. Use --noexit option to keep UI running after failure.\n")
		}
		if serverUpdate != nil {
			<-serverUpdate
		}
		util.Suicide(false)
	}
}
