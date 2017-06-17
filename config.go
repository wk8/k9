package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type Config struct {
	root *configNode
	// we cache the result of resolved metrics for efficiency
	resolvedMetrics map[string]*ConfigValue
}

type configNode struct {
	children map[string]*configNode
	value    *ConfigValue
}

type ConfigValue struct {
	remove       bool
	tagsToRemove map[string]bool
}

func (config *Config) ConfigFor(metric string) *ConfigValue {
	configValue := config.resolvedMetrics[metric]
	if configValue != nil {
		// already cached
		// TODO wkpo unit tests on the caching?
		return configValue
	}

	configValue = newConfigValue()
	resolveConfigFor(strings.Split(metric, "."), 0, config.root, configValue, false)
	if configValue.remove {
		// no need to keep tags in memory
		// TODO wkpo unit test on this too!
		configValue.tagsToRemove = nil
	}

	config.resolvedMetrics[metric] = configValue
	return configValue
}

func resolveConfigFor(path []string, currentIndex int, currentNode *configNode,
	configValue *ConfigValue, ongoingDoubleWildcard bool) {

	if currentNode == nil {
		return
	}

	if currentIndex >= len(path) {
		configValue.merge(currentNode.value)
		return
	}

	// double wildcards are often used to remove, so let's do them first
	resolveConfigFor(path, currentIndex+1, currentNode.children["**"], configValue, true)
	if configValue.remove {
		return
	}
	if ongoingDoubleWildcard {
		resolveConfigFor(path, currentIndex+1, currentNode, configValue, true)
		if configValue.remove {
			return
		}
	}

	// then, single wildcard
	resolveConfigFor(path, currentIndex+1, currentNode.children["*"], configValue, false)
	if configValue.remove {
		return
	}

	// then exact match
	resolveConfigFor(path, currentIndex+1, currentNode.children[path[currentIndex]], configValue, false)
}

type configFileContent struct {
	Remove_metrics []string
	Remove_tags    []struct {
		Metrics []string
		Tags    []string
	}
}

func NewConfig() (config *Config) {
	return &Config{
		root:            newConfigNode(),
		resolvedMetrics: make(map[string]*ConfigValue),
	}
}

func (config *Config) mergeFromFile(filename string) error {
	rawContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	content := configFileContent{}
	err = yaml.Unmarshal(rawContent, &content)
	if err != nil {
		return err
	}

	config.merge(&content)

	return nil
}

func (config *Config) merge(content *configFileContent) {
	for _, metric := range content.Remove_metrics {
		config.mergeMetric(metric, &ConfigValue{remove: true})
	}

	for _, metricsAndTags := range content.Remove_tags {
		for _, metric := range metricsAndTags.Metrics {
			tagsToRemove := make(map[string]bool)
			for _, tag := range metricsAndTags.Tags {
				tagsToRemove[tag] = true
			}
			config.mergeMetric(metric, &ConfigValue{tagsToRemove: tagsToRemove})
		}
	}
}

func (config *Config) mergeMetric(metric string, value *ConfigValue) {
	currentNode := config.root
	for _, key := range strings.Split(metric, ".") {
		newNode := currentNode.children[key]

		if newNode == nil {
			newNode = newConfigNode()
			currentNode.children[key] = newNode
		}

		currentNode = newNode
	}

	if currentNode.value == nil {
		currentNode.value = newConfigValue()
	}
	currentNode.value.merge(value)
}

func newConfigNode() *configNode {
	return &configNode{children: make(map[string]*configNode)}
}

func newConfigValue() *ConfigValue {
	return &ConfigValue{tagsToRemove: make(map[string]bool)}
}

func (value *ConfigValue) merge(other *ConfigValue) {
	if other == nil {
		return
	}

	value.remove = value.remove || other.remove
	if other.tagsToRemove != nil {
		for tag, _ := range other.tagsToRemove {
			value.tagsToRemove[tag] = true
		}
	}
}
