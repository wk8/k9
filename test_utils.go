package main

import (
	"bytes"
	"log"
	"os"
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

var logLineRegex = regexp.MustCompile("^[0-9]{4}(?:/[0-9]{2}){2} [0-9]{2}(?::[0-9]{2}){2} (?P<Rest>.*)$")

func CheckLogLines(t *testing.T, output string, expectedLines []string) bool {
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
