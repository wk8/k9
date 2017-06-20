package main

//*/
import (
	"os"
	"os/signal"
)

func main() {
	proxy := NewProxy("https://app.datadoghq.com", nil)
	// proxy := NewProxy("http://localhost:8181", nil)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	proxy.Start(8283)
	<-stop
	proxy.Stop()
}

//*/

/*/
func main() {
	config := NewPruningConfig()
	config.MergeWithFile("test_fixtures/config.yml")
}

//*/
