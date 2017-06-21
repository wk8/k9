package main

import (
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
  PruningConfig *PruningConfig
  
	path          string
	logLevelSet   bool
}

const DEFAULT_K9_CONFIG_PATH = "/etc/k9/k9.conf"

// TODO wkpo take an optional logLevel too, and TODO wkpo doc!!
func NewConfig(args ...string) *Config {
	path := DEFAULT_K9_CONFIG_PATH

	switch len(args) {
	case 0:
	case 1:
		path = args[0]
	default:
		logFatal("Too many arguments for NewConfig")
	}

	config := &Config{path: path}
	config.load(true)
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
		logFatal("Unable to read the config at %v: %v", filename, err)
	}

	content := configFileContent{}
	err = yaml.Unmarshal(rawContent, &content)
	if err != nil {
		logFatal("Unable to parse the config at %v: %v", filename, err)
	}

	if !config.logLevelSet {
		_, err = setLogLevelFromString(content.Log_level)
		config.logLevelSet = err == nil
	}

	config.loadPruningConfig(content.Pruning_configs, initialLoad)
}

func (config *Config) loadPruningConfig(pruningConfigsPaths []string, initialLoad bool) {
	newPruningConfig := NewPruningConfig()
  atLeastOneMerged := false

  for _, pruningConfigPath := range pruningConfigsPaths {
    err := func newPruningConfig.MergeWithFile(pruningConfigPath)
    
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
