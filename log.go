package main

import "log"

// TODO wkpo make this better... at least a level?

func logError(format string, v ...interface{}) {
	doLog("ERROR", format, v...)
}

func doLog(level string, format string, v ...interface{}) {
	log.Printf(level+": "+format, v...)
}
