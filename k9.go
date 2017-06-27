package main

import (
	"flag"
	"fmt"
)

var VERSION string

const DEFAULT_CONFIG_PATH = "/etc/k9/k9.conf"

func main() {
	// parse command line args
	configPath := flag.String("c", DEFAULT_CONFIG_PATH, "The path to the k9 configuration")
	logLevel := flag.String("l", "", "The severity level to use for logging (must be one of DEBUG, INFO, WARN, ERROR, FATAL - if present, overrides the one defined in the config file, if any - otherwise defaults to INFO)")
	isDebug := flag.Bool("d", false, "Debug mode, equivalent to -l DEBUG")
	isVersion := flag.Bool("v", false, "Outputs the version then exits")
	flag.Parse()

	if *isVersion {
		fmt.Println("Version:", VERSION)
		return
	}

	// create the config
	if *isDebug {
		*logLevel = "DEBUG"
	}
	config := NewConfig(*configPath, *logLevel)

	// start the proxy
	proxy := NewProxy(config.DdUrl, &DDTransformer{config: config.PruningConfig})
	proxy.Start(config.ListenPort)

	// then listen for signals
	reloaderShutdowner := &k9ReloaderShutdowner{
		config: config,
		proxy:  proxy,
	}

	signalListener := &SignalListener{reloaderShutdowner: reloaderShutdowner}
	signalListener.Run()
}

type k9ReloaderShutdowner struct {
	config *Config
	proxy  *HttpProxy
}

func (reloaderShutdowner *k9ReloaderShutdowner) Reload() {
	reloaderShutdowner.config.Reload()
}

func (reloaderShutdowner *k9ReloaderShutdowner) Shutdown() {
	reloaderShutdowner.proxy.Stop()
}
