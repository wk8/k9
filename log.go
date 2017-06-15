package main

import (
	"log"
	"os"
)

// TODO wkpo make this better... at least set a level?

func logDebug(format string, v ...interface{}) {
	doLog("DEBUG", format, v...)
}

func logInfo(format string, v ...interface{}) {
	doLog("INFO", format, v...)
}

func logError(format string, v ...interface{}) {
	doLog("ERROR", format, v...)
}

func logFatal(format string, v ...interface{}) {
	doLog("FATAL", format, v...)
	os.Exit(1)
}

func doLog(level string, format string, v ...interface{}) {
	log.Printf(level+": "+format, v...)
}
