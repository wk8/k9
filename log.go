package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

const DEFAULT_LOG_LEVEL = INFO

var logLevel LogLevel = DEFAULT_LOG_LEVEL

func setLogLevel(newLevel LogLevel) (previousLevel LogLevel) {
	previousLevel = logLevel
	logLevel = newLevel
	return
}

func setLogLevelFromString(newLevelAsStr string) (LogLevel, error) {
	var newLevel LogLevel = -1

	switch strings.ToUpper(newLevelAsStr) {
	case "DEBUG":
		newLevel = DEBUG
	case "INFO":
		newLevel = INFO
	case "WARN":
		newLevel = WARN
	case "ERROR":
		newLevel = ERROR
	case "FATAL":
		newLevel = FATAL
	}

	if newLevel == -1 {
		var buffer bytes.Buffer
		fmt.Fprintf(&buffer, "Unknown log level, ignoring: %v", newLevelAsStr)
		message := buffer.String()
		logWarn(message)
		return logLevel, errors.New(message)
	} else {
		return setLogLevel(newLevel), nil
	}
}

// comes in handy to log expensive operations in debug mode
func logDebugWith(format string, callback func() []interface{}) {
	if logLevel <= DEBUG {
		logDebug(format, callback()...)
	}
}

func logDebug(format string, v ...interface{}) {
	if logLevel <= DEBUG {
		doLog("DEBUG", format, v...)
	}
}

func logInfo(format string, v ...interface{}) {
	if logLevel <= INFO {
		doLog("INFO", format, v...)
	}
}

func logWarn(format string, v ...interface{}) {
	if logLevel <= WARN {
		doLog("WARN", format, v...)
	}
}

func logError(format string, v ...interface{}) {
	if logLevel <= ERROR {
		doLog("ERROR", format, v...)
	}
}

func logFatal(format string, v ...interface{}) {
	doLog("FATAL", format, v...)
	os.Exit(1)
}

func doLog(level string, format string, v ...interface{}) {
	log.Printf(level+": "+format, v...)
}
