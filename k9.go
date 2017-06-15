package main

import (
	"os"
	"os/signal"
)

func main() {
	proxy := NewProxy("http://localhost:8181", nil)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	proxy.Start(8081)
	<-stop
	proxy.Stop()
}
