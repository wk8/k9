package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	PruningConfig  *PruningConfig
	ListenPort     int
	DdUrl          string
	ApiKey         string
	ApplicationKey string

	path        string
	logLevelSet bool
}

func NewConfig(path, logLevel string) *Config {
	config := &Config{
		PruningConfig: NewPruningConfig(),
		ListenPort:    8283,
		DdUrl:         "https://app.datadoghq.com",
		path:          path,
	}
	config.maybeSetLogLevel(logLevel)
	config.load(true)

	return config
}

func (config *Config) maybeSetLogLevel(newLevel string) {
	if newLevel == "" || config.logLevelSet {
		return
	}

	_, err := setLogLevelFromString(newLevel)
	config.logLevelSet = err == nil
}

type configFileContent struct {
	Log_level       string
	Dd_Url          string
	Listen_port     int
	Api_key         string
	Application_key string
	Pruning_configs []string
}

func (config *Config) Reload() {
	logInfo("Reloading configuration...")
	config.load(false)
}

func (config *Config) load(initialLoad bool) {
	rawContent, err := ioutil.ReadFile(config.path)
	if err != nil {
		logFatal("Unable to read the config at %v: %v", config.path, err)
	}

	content := configFileContent{}
	err = yaml.Unmarshal(rawContent, &content)
	if err != nil {
		logFatal("Unable to parse the config at %v: %v", config.path, err)
	}

	config.maybeSetLogLevel(content.Log_level)
	config.loadPruningConfig(content.Pruning_configs, initialLoad)

	if initialLoad {
		if content.Listen_port > 0 {
			config.ListenPort = content.Listen_port
		}
		if content.Dd_Url != "" {
			config.DdUrl = content.Dd_Url
		}
		config.ApiKey = content.Api_key
		config.ApplicationKey = content.Application_key
	}
}

func (config *Config) loadPruningConfig(pruningConfigsPaths []string, initialLoad bool) {
	newPruningConfig := NewPruningConfig()

	for _, pruningConfigPath := range pruningConfigsPaths {
		newPruningConfig.MergeWithFileOrGlob(pruningConfigPath)
	}

	config.PruningConfig.Reset(newPruningConfig)
}
