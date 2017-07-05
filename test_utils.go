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

func WithCatpuredLogging(fun func()) string {
	var buffer bytes.Buffer
	log.SetOutput(&buffer)
	fun()
	log.SetOutput(os.Stderr)
	return buffer.String()
}

func WithLogLevelAndCapturedLogging(logLevel LogLevel, fun func()) string {
	previousLogLevel := setLogLevel(logLevel)
	output := WithCatpuredLogging(fun)
	setLogLevel(previousLogLevel)
	return output
}

var logLineRegex = regexp.MustCompile("^[0-9]{4}(?:/[0-9]{2}){2} [0-9]{2}(?::[0-9]{2}){2} (?P<Rest>.*)$")

func CheckLogLines(t *testing.T, output string, expectedLines ...string) bool {
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

// idea stolen from
// https://stackoverflow.com/questions/26225513/how-to-test-os-exit-scenarios-in-go
// returns the output
func AssertCrashes(t *testing.T, testCaseName string, testCase func()) string {
	if os.Getenv("K9_ASSERT_CRASHES") == "1" {
		testCase()
		return "<DID NOT CRASH!>"
	}

	cmd := exec.Command(os.Args[0], "-test.run="+testCaseName)
	cmd.Env = append(os.Environ(), "K9_ASSERT_CRASHES=1")
	var buffer bytes.Buffer
	cmd.Stderr = &buffer

	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); !ok || e.Success() {
		t.Errorf("Test case %v with err %v, want exit status 1", testCaseName, err)
	}
	return buffer.String()
}

func IsCircle() bool {
	_, isCircle := os.LookupEnv("CIRCLECI")
	return isCircle
}
