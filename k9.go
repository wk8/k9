package main

import (
	"flag"
	// "os/signal"
)

const DEFAULT_K9_CONFIG_PATH = "/etc/k9/k9.conf"
const DEFAULT_LISTEN_PORT = 8283
const DEFAULT_DD_URL = "https://app.datadoghq.com"

func main() {
	// parse command line args
	configPath := flag.String("c", DEFAULT_K9_CONFIG_PATH, "The path to the k9 configuration")
	listenPort := flag.Int("p", DEFAULT_LISTEN_PORT, "The port on which k9 will listen locally")
	logLevel := flag.String("l", "", "The severity level to use for logging (must be one of DEBUG, INFO, WARN, ERROR, FATAL, default INFO)")
	isDebug := flag.Bool("d", false, "Debug mode, equivalent to -l DEBUG")
	ddUrl := flag.String("u", DEFAULT_DD_URL, "The Datadog endpoint to hit")
	flag.Parse()

	// create the config
	if isDebug {
		logLevel = "DEBUG"
	}
	config := NewConfig(configPath, logLevel)

	// start the proxy
	proxy := NewProxy(ddUrl, &DDTransformer{})
	proxy.Start(listenPort)

	// and then wait for reload signals!

}

/*/
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

// TODO wkpo next https://github.com/golang/go/issues/10377 to read from k9-917742246
