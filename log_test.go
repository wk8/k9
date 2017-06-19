package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func TestLogLevels(t *testing.T) {
	t.Run("debug", func(t *testing.T) {
		output := withLogLevelAndCapturedLogging(DEBUG, func() {
			logDebug("%v %v - %v", "coucou", "toi", 28)
			logInfo("hey you")
			logWarn("getting lonely")
			logError("getting old")
		})

		if !checkLogLines(t, output, []string{
			"DEBUG: coucou toi - 28",
			"INFO: hey you",
			"WARN: getting lonely",
			"ERROR: getting old"}) {
			t.Errorf("Unexpected output: %v", output)
		}
	})

	t.Run("info", func(t *testing.T) {
		output := withLogLevelAndCapturedLogging(INFO, func() {
			logDebug("%v %v - %v", "coucou", "toi", 28)
			logInfo("hey you")
			logWarn("getting lonely")
			logError("getting old")
		})

		if !checkLogLines(t, output, []string{
			"INFO: hey you",
			"WARN: getting lonely",
			"ERROR: getting old"}) {
			t.Errorf("Unexpected output: %v", output)
		}
	})

	t.Run("warn", func(t *testing.T) {
		output := withLogLevelAndCapturedLogging(WARN, func() {
			logDebug("%v %v - %v", "coucou", "toi", 28)
			logInfo("hey you")
			logWarn("getting lonely")
			logError("getting old")
		})

		if !checkLogLines(t, output, []string{
			"WARN: getting lonely",
			"ERROR: getting old"}) {
			t.Errorf("Unexpected output: %v", output)
		}
	})

	t.Run("error", func(t *testing.T) {
		output := withLogLevelAndCapturedLogging(ERROR, func() {
			logDebug("%v %v - %v", "coucou", "toi", 28)
			logInfo("hey you")
			logWarn("getting lonely")
			logError("getting old")
		})

		if !checkLogLines(t, output, []string{"ERROR: getting old"}) {
			t.Errorf("Unexpected output: %v", output)
		}
	})
}

func TestLogFatal(t *testing.T) {
	// idea stolen from
	// https://stackoverflow.com/questions/26225513/how-to-test-os-exit-scenarios-in-go
	if os.Getenv("K9_TEST_LOG_FATAL") == "1" {
		logFatal("hey teacher, leave those kids alone")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestLogFatal")
	cmd.Env = append(os.Environ(), "K9_TEST_LOG_FATAL=1")
	var buffer bytes.Buffer
	cmd.Stderr = &buffer

	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", err)
	}
	output := buffer.String()
	if !checkLogLines(t, output, []string{"FATAL: hey teacher, leave those kids alone"}) {
		t.Errorf("Unexpected output: %v", output)
	}
}

// Private helpers

func withCatpuredLogging(fun func()) string {
	var buffer bytes.Buffer
	log.SetOutput(&buffer)
	fun()
	log.SetOutput(os.Stderr)
	return buffer.String()
}

func withLogLevelAndCapturedLogging(logLevel LogLevel, fun func()) string {
	previousLogLevel := setLogLevel(logLevel)
	output := withCatpuredLogging(fun)
	setLogLevel(previousLogLevel)
	return output
}

var logLineRegex = regexp.MustCompile("^[0-9]{4}(?:/[0-9]{2}){2} [0-9]{2}(?::[0-9]{2}){2} (?P<Rest>.*)$")

func checkLogLines(t *testing.T, output string, expectedLines []string) bool {
	output = strings.TrimSpace(output)
	actualLines := strings.Split(output, "\n")

	if len(expectedLines) != len(actualLines) {
		t.Errorf("Different number of lines: %v VS %v in %#v", len(expectedLines), len(actualLines), actualLines)
		return false
	}

	for i, line := range expectedLines {
		match := logLineRegex.FindStringSubmatch(actualLines[i])

		if match == nil || match[1] != line {
			t.Errorf("Unexpected line at position %v: %v", i, actualLines[i])
			return false
		}
	}

	return true
}