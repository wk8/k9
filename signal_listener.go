package main

import (
	"os"
	"os/signal"
	"syscall"
)

type ReloaderShutdowner interface {
	Reload()
	Shutdown()
}

type SignalListener struct {
	reloaderShutdowner ReloaderShutdowner
}

// blocking call
func (listener *SignalListener) Run() {
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	for {
		signal := <-channel

		switch signal {
		case syscall.SIGHUP:
			logInfo("Received SIGHUP, reloading")
			listener.reloaderShutdowner.Reload()
		case syscall.SIGINT:
			fallthrough
		case syscall.SIGTERM:
			logInfo("Shutting down...")
			listener.reloaderShutdowner.Shutdown()
			return
		}
	}
}
