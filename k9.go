package main

//*/
import (
	"os"
	"os/signal"
)

func main() {
	proxy := NewProxy("https://app.datadoghq.com", &DDTransformer{})
	// proxy := NewProxy("http://localhost:8181", &DDTransformer{})
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

// TODO wkpo next https://github.com/golang/go/issues/10377 to read from k9-917742246
