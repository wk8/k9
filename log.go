package main

import (
	"log"
	"os"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var logLevel LogLevel = DEBUG // TODO wkpo INFO

func setLogLevel(newLevel LogLevel) (previousLevel LogLevel) {
	previousLevel = logLevel
	logLevel = newLevel
	return
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
