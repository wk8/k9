package main

import (
	"os"
	"os/signal"
)

func main() {
	proxy := NewProxy(nil)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	proxy.Start()
	<-stop
	proxy.Stop()
}
