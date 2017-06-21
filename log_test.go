package main

import (
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

		if !CheckLogLines(t, output, []string{
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

		if !CheckLogLines(t, output, []string{
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

		if !CheckLogLines(t, output, []string{
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

		if !CheckLogLines(t, output, []string{"ERROR: getting old"}) {
			t.Errorf("Unexpected output: %v", output)
		}
	})
}

func TestLogFatal(t *testing.T) {
	output := AssertCrashes(t, func() {
		logFatal("hey teacher, leave those kids alone")
	}, "TestLogFatal")

	if !CheckLogLines(t, output, []string{"FATAL: hey teacher, leave those kids alone"}) {
		t.Errorf("Unexpected output: %v", output)
	}
}

func TestLogDebugWith(t *testing.T) {
	t.Run("when the log level is set to debug, it uses the callback to log",
		func(t *testing.T) {
			output := withLogLevelAndCapturedLogging(DEBUG, func() {
				logDebugWith("%v %v - %v", func() []interface{} {
					return []interface{}{"coucou", "toi", 28}
				})
			})

			if !CheckLogLines(t, output, []string{"DEBUG: coucou toi - 28"}) {
				t.Errorf("Unexpected output: %v", output)
			}
		})

	t.Run("when the log level is not set to debug, it doesn't run the callback",
		func(t *testing.T) {
			output := withLogLevelAndCapturedLogging(INFO, func() {
				logDebugWith("%v %v - %v", func() []interface{} {
					t.Fatalf("The callback is being called")
					return []interface{}{}
				})
			})

			if output != "" {
				t.Errorf("Unexpected output: %v", output)
			}
		})
}

func TestSetLogLevelFromString(t *testing.T) {
	previousLogLevel := logLevel

	t.Run("it successfully parses and sets the level when fed a correct level",
		func(t *testing.T) {
			output := WithCatpuredLogging(func() {
				setLogLevelFromString("DEBUG")
			})

			if logLevel != DEBUG {
				t.Errorf("Unexpected log level: %v", logLevel)
			}
			if output != "" {
				t.Errorf("Unexpected output: %v", output)
			}

			output = WithCatpuredLogging(func() {
				setLogLevelFromString("ERROR")
			})

			if logLevel != ERROR {
				t.Errorf("Unexpected log level: %v", logLevel)
			}
			if output != "" {
				t.Errorf("Unexpected output: %v", output)
			}
		})

	t.Run("it is not case sensitive",
		func(t *testing.T) {
			output := WithCatpuredLogging(func() {
				setLogLevelFromString("wARn")
			})

			if logLevel != WARN {
				t.Errorf("Unexpected log level: %v", logLevel)
			}
			if output != "" {
				t.Errorf("Unexpected output: %v", output)
			}
		})

	t.Run("if given an incorrect log level, returns an error and outputs a warning",
		func(t *testing.T) {
			var err error
			output := WithCatpuredLogging(func() {
				_, err = setLogLevelFromString("hey")
			})

			if !CheckLogLines(t, output, []string{"WARN: Unknown log level, ignoring: hey"}) {
				t.Errorf("Unexpected output: %v", output)
			}
			if err == nil {
				t.Errorf("Should have errored out")
			}
		})

	setLogLevel(previousLogLevel)
}

// Private helpers

func withLogLevelAndCapturedLogging(logLevel LogLevel, fun func()) string {
	previousLogLevel := setLogLevel(logLevel)
	output := WithCatpuredLogging(fun)
	setLogLevel(previousLogLevel)
	return output
}
