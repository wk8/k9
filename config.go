package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	PruningConfig *PruningConfig

	path        string
	logLevelSet bool
}

const DEFAULT_K9_CONFIG_PATH = "/etc/k9/k9.conf"

func NewConfig(path, logLevel string) *Config {
	if path == "" {
		path = DEFAULT_K9_CONFIG_PATH
	}

	config := &Config{path: path}
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
	Pruning_configs []string
}

func (config *Config) Reload() {
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
}

func (config *Config) loadPruningConfig(pruningConfigsPaths []string, initialLoad bool) {
	newPruningConfig := NewPruningConfig()
	atLeastOneMerged := false

	for _, pruningConfigPath := range pruningConfigsPaths {
		err := newPruningConfig.MergeWithFile(pruningConfigPath)

		if err == nil {
			atLeastOneMerged = true
		} else {
			logWarn("Unable to load pruning config from %v: %v", pruningConfigPath, err)
		}
	}

	if atLeastOneMerged {
		config.PruningConfig = newPruningConfig
	} else {
		if initialLoad {
			logFatal("No pruning config file loaded; please fix your configuration, exiting")
		} else {
			logWarn("No pruning config file loaded, keeping the previous pruning configuration")
		}
	}
}
