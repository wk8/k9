package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"gopkg.in/yaml.v2"
)

type PruningConfig struct {
	root *configNode
	// we cache the results for resolved metrics for efficiency
	resolvedMetrics map[string]*MetricPruningConfig
}

type configNode struct {
	children map[string]*configNode
	value    *configValue
}

type configValue struct {
	remove     bool
	keep       bool
	removeTags map[string]bool
	keepTags   map[string]bool
}

type MetricPruningConfig struct {
	Remove     bool
	RemoveTags map[string]bool
}

func NewPruningConfig() (config *PruningConfig) {
	return &PruningConfig{
		root:            newConfigNode(),
		resolvedMetrics: make(map[string]*MetricPruningConfig),
	}
}

func (config *PruningConfig) Reset(other *PruningConfig) {
	config.root = other.root
	config.resolvedMetrics = make(map[string]*MetricPruningConfig)
}

func (config *PruningConfig) ConfigFor(metric string) *MetricPruningConfig {
	metricPruningConfig := config.resolvedMetrics[metric]

	if metricPruningConfig == nil {
		// not cached yet
		configValue := newConfigValue()
		resolveConfigFor(strings.Split(metric, "."), 0, config.root, configValue, false)

		metricPruningConfig = configValue.toMetricPruningConfig()
	}

	config.resolvedMetrics[metric] = metricPruningConfig
	return metricPruningConfig
}

func resolveConfigFor(path []string, currentIndex int, currentNode *configNode,
	configValue *configValue, ongoingDoubleWildcard bool) {

	if currentNode == nil {
		return
	}

	if currentIndex >= len(path) {
		configValue.merge(currentNode.value)
		return
	}

	// double wildcards
	resolveConfigFor(path, currentIndex+1, currentNode.children["**"], configValue, true)
	if ongoingDoubleWildcard {
		resolveConfigFor(path, currentIndex+1, currentNode, configValue, true)
	}

	// then, single wildcard
	resolveConfigFor(path, currentIndex+1, currentNode.children["*"], configValue, false)

	// then exact match
	resolveConfigFor(path, currentIndex+1, currentNode.children[path[currentIndex]], configValue, false)
}

type pruningConfigFileContentTagsConfig struct {
	Metrics []string
	Tags    []string
}

type pruningConfigFileContent struct {
	Metrics struct {
		Remove []string
		Keep   []string
	}

	Tags struct {
		Remove []pruningConfigFileContentTagsConfig
		Keep   []pruningConfigFileContentTagsConfig
	}
}

func (config *PruningConfig) MergeWithFileOrGlob(filenameOrGlob string) {
	err := config.mergeWithFile(filenameOrGlob)

	if pathError, ok := err.(*os.PathError); ok && pathError.Err == syscall.ENOENT {
		// maybe it's a glob?
		matches, globErr := filepath.Glob(filenameOrGlob)

		if globErr == nil && len(matches) > 0 {
			err = nil
			for _, filename := range matches {
				if newErr := config.mergeWithFile(filename); newErr != nil {
					logWarn("Unable to load pruning config from %v: %v", filename, newErr)
				}
			}
		}
	}

	if err != nil {
		logWarn("Unable to load pruning config from %v: %v", filenameOrGlob, err)
	}
}

func (config *PruningConfig) mergeWithFile(filename string) error {
	rawContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	content := pruningConfigFileContent{}
	err = yaml.Unmarshal(rawContent, &content)
	if err != nil {
		return err
	}

	config.merge(&content)

	return nil
}

func (config *PruningConfig) merge(content *pruningConfigFileContent) {
	// metrics
	for _, metric := range content.Metrics.Remove {
		config.mergeNode(metric, &configValue{remove: true})
	}
	for _, metric := range content.Metrics.Keep {
		config.mergeNode(metric, &configValue{keep: true})
	}

	// tags
	config.mergeTags(content.Tags.Remove, false)
	config.mergeTags(content.Tags.Keep, true)
}

func (config *PruningConfig) mergeTags(tagsConfigs []pruningConfigFileContentTagsConfig, keep bool) {
	for _, metricsAndTags := range tagsConfigs {
		for _, metric := range metricsAndTags.Metrics {
			tags := make(map[string]bool)
			for _, tag := range metricsAndTags.Tags {
				tags[tag] = true
			}

			var value configValue
			if keep {
				value = configValue{keepTags: tags}
			} else {
				value = configValue{removeTags: tags}
			}

			config.mergeNode(metric, &value)
		}
	}
}

func (config *PruningConfig) mergeNode(metric string, value *configValue) {
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

func (value *configValue) merge(other *configValue) {
	if other == nil {
		return
	}

	value.remove = value.remove || other.remove
	value.keep = value.keep || other.keep

	if other.removeTags != nil {
		for tag, _ := range other.removeTags {
			value.removeTags[tag] = true
		}
	}
	if other.keepTags != nil {
		for tag, _ := range other.keepTags {
			value.keepTags[tag] = true
		}
	}
}

func (configValue *configValue) toMetricPruningConfig() *MetricPruningConfig {
	if configValue.remove && !configValue.keep {
		return &MetricPruningConfig{Remove: true}
	} else {
		removeTags := make(map[string]bool)

		for tag, _ := range configValue.removeTags {
			if !configValue.keepTags[tag] {
				removeTags[tag] = true
			}
		}

		return &MetricPruningConfig{RemoveTags: removeTags}
	}
}

func newConfigNode() *configNode {
	return &configNode{children: make(map[string]*configNode)}
}

func newConfigValue() *configValue {
	return &configValue{
		removeTags: make(map[string]bool),
		keepTags:   make(map[string]bool),
	}
}
