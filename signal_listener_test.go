package main

import (
	"bytes"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestWithSigInt(t *testing.T) {
	output := signalListenerTest(t, "TestWithSigInt", syscall.SIGHUP, syscall.SIGHUP, syscall.SIGINT)

	if !CheckLogLines(t, output,
		"INFO: Received SIGHUP, reloading",
		"WARN: testReloaderShutdowner reloading: 1",
		"INFO: Received SIGHUP, reloading",
		"WARN: testReloaderShutdowner reloading: 2",
		"INFO: Shutting down...",
		"WARN: testReloaderShutdowner shutting down") {
		t.Errorf("Unexpected output: %#v", output)
	}
}

func TestWithSigTerm(t *testing.T) {
	output := signalListenerTest(t, "TestWithSigInt", syscall.SIGTERM)

	if !CheckLogLines(t, output,
		"INFO: Shutting down...",
		"WARN: testReloaderShutdowner shutting down") {
		t.Errorf("Unexpected output: %#v", output)
	}
}

/// Private helpers

// returns the output, after sending the given signals
func signalListenerTest(t *testing.T, testCaseName string, signals ...os.Signal) string {
	if os.Getenv("K9_SIGNAL_LISTENER_TEST") == "1" {
		reloaderShutdowner := &testReloaderShutdowner{}
		signalListener := &SignalListener{reloaderShutdowner: reloaderShutdowner}
		signalListener.Run()
		os.Exit(0)
	}

	cmd := exec.Command(os.Args[0], "-test.run="+testCaseName)
	cmd.Env = append(os.Environ(), "K9_SIGNAL_LISTENER_TEST=1")
	var buffer bytes.Buffer
	cmd.Stderr = &buffer

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	childPid := cmd.Process.Pid
	child, err := os.FindProcess(childPid)
	if err != nil {
		t.Fatal(err)
	}

	for _, signal := range signals {
		// needed to let the child process signals
		time.Sleep(10 * time.Millisecond)
		child.Signal(signal)
	}

	// wait for the child to die, but not for too long
	timeout := 5 * time.Second
	timedOut := make(chan bool, 1)
	go func() {
		cmd.Wait()
		timedOut <- false
	}()
	go func() {
		time.Sleep(timeout)
		timedOut <- true
	}()

	if <-timedOut {
		// let's try and kill it anyway
		child.Signal(syscall.SIGKILL)
		t.Fatalf("Waited for %v for the child (PID %v) to exit (output so far: %v)",
			timeout, childPid, buffer.String())
	}

	return buffer.String()
}

type testReloaderShutdowner struct {
	reloadedCount int
}

func (reloaderShutdowner *testReloaderShutdowner) Reload() {
	reloaderShutdowner.reloadedCount++
	logWarn("testReloaderShutdowner reloading: %v", reloaderShutdowner.reloadedCount)
}

func (reloaderShutdowner *testReloaderShutdowner) Shutdown() {
	logWarn("testReloaderShutdowner shutting down")
}
