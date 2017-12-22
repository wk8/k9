package main

import (
	"reflect"
	"strconv"
	"testing"
)

func TestBaseConfig(t *testing.T) {
	config := NewPruningConfig()
	config.MergeWithFileOrGlob("test_fixtures/pruning_configs/full.yml")

	// my_app.elasticsearch.count
	pruningConfig := config.ConfigFor("my_app.elasticsearch.count")
	expectedPruningConfig := &MetricPruningConfig{
		Remove:     false,
		RemoveTags: map[string]bool{"hide_this": true, "role": true, "instance-type": true, "es_host": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// my_app.elasticsearch.time.95percentile
	pruningConfig = config.ConfigFor("my_app.elasticsearch.time.95percentile")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:     false,
		RemoveTags: map[string]bool{"hide_this": true, "role": true, "instance-type": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// my_app.elasticsearch.time.max
	pruningConfig = config.ConfigFor("my_app.elasticsearch.time.max")
	expectedPruningConfig = &MetricPruningConfig{Remove: true}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// my_app.elasticsearch.time.min
	pruningConfig = config.ConfigFor("my_app.elasticsearch.time.min")
	expectedPruningConfig = &MetricPruningConfig{Remove: true}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// my_app.profile.my_app_partner_order.distribution.time.95percentile
	pruningConfig = config.ConfigFor("my_app.profile.my_app_partner_order.distribution.time.95percentile")
	expectedPruningConfig = &MetricPruningConfig{Remove: true}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// my_app.profile.something.95percentile
	pruningConfig = config.ConfigFor("my_app.profile.something.95percentile")
	expectedPruningConfig = &MetricPruningConfig{Remove: true}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// top_level_metric
	pruningConfig = config.ConfigFor("top_level_metric")
	expectedPruningConfig = &MetricPruningConfig{Remove: true}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// another_top_level_metric
	pruningConfig = config.ConfigFor("another_top_level_metric")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:     false,
		RemoveTags: map[string]bool{"not_for_top_level_metrics": true, "whatever": true, "hide_this": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// i_dont_appear_in_the_config
	pruningConfig = config.ConfigFor("i_dont_appear_in_the_config")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:     false,
		RemoveTags: map[string]bool{"not_for_top_level_metrics": true, "hide_this": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// my_app.profile.something.avg
	pruningConfig = config.ConfigFor("my_app.profile.something.avg")
	expectedPruningConfig = &MetricPruningConfig{Remove: true}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// my_app.profile.something.something.avg
	pruningConfig = config.ConfigFor("my_app.profile.something.something.avg")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:     false,
		RemoveTags: map[string]bool{"hide_this": true, "role": true, "instance-type": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// my_app.hey.there
	pruningConfig = config.ConfigFor("my_app.hey.there")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:     false,
		RemoveTags: map[string]bool{"role": true, "instance-type": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// my_app.profile.some.important.function.95percentile
	pruningConfig = config.ConfigFor("my_app.profile.some.important.function.95percentile")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:     false,
		RemoveTags: map[string]bool{"role": true, "instance-type": true, "hide_this": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}
}

func TestCaching(t *testing.T) {
	config := NewPruningConfig()
	config.MergeWithFileOrGlob("test_fixtures/pruning_configs/full.yml")

	// the cache should be empty
	expectedResolvedMetrics := map[string]*MetricPruningConfig{}
	if !reflect.DeepEqual(config.resolvedMetrics, expectedResolvedMetrics) {
		t.Errorf("Unexpected cache: %#v", config.resolvedMetrics)
	}

	pruningConfig := config.ConfigFor("my_app.test_caching")
	expectedPruningConfig := &MetricPruningConfig{
		Remove:     false,
		RemoveTags: map[string]bool{"role": true, "instance-type": true, "hide_this": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// now it should be in the cache
	expectedResolvedMetrics["my_app.test_caching"] = expectedPruningConfig
	if !reflect.DeepEqual(config.resolvedMetrics, expectedResolvedMetrics) {
		t.Errorf("Unexpected cache: %#v", config.resolvedMetrics)
	}

	// calling a second time should yield the same value
	pruningConfig = config.ConfigFor("my_app.test_caching")
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}
}

func TestSeveralFiles(t *testing.T) {
	configFromFull := NewPruningConfig()
	configFromFull.MergeWithFileOrGlob("test_fixtures/pruning_configs/full.yml")

	configFromPartials := NewPruningConfig()
	for i := 1; i <= 4; i++ {
		configFromPartials.MergeWithFileOrGlob("test_fixtures/pruning_configs/" + strconv.Itoa(i) + ".yml")
	}

	if !reflect.DeepEqual(configFromFull, configFromPartials) {
		t.Errorf("Unexpectedly different configs:\n%#v\nVS\n%#v", configFromFull, configFromPartials)
		// the above doesn't yield usable output when failing...
		compareConfigTrees(t, configFromFull.root, configFromPartials.root, "")
	}
}

func TestGlob(t *testing.T) {
	configFromFull := NewPruningConfig()
	configFromFull.MergeWithFileOrGlob("test_fixtures/pruning_configs/full.yml")

	configFromGlob := NewPruningConfig()
	configFromGlob.MergeWithFileOrGlob("test_fixtures/pruning_configs/[1-4].yml")

	if !reflect.DeepEqual(configFromFull, configFromGlob) {
		t.Errorf("Unexpectedly different configs:\n%#v\nVS\n%#v", configFromFull, configFromGlob)
		// the above doesn't yield usable output when failing...
		compareConfigTrees(t, configFromFull.root, configFromGlob.root, "")
	}
}

func TestHostTags(t *testing.T) {
	config := NewPruningConfig()
	config.MergeWithFileOrGlob("test_fixtures/pruning_configs/host_tags.yml")

	pruningConfig := config.ConfigFor("my_app.my_metric")
	expectedPruningConfig := &MetricPruningConfig{
		Remove:       false,
		RemoveTags:   map[string]bool{"host": true, "instance-type": true},
		RemoveHost:   true,
		KeepHostTags: false,
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	pruningConfig = config.ConfigFor("my_app.special")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:       false,
		RemoveTags:   map[string]bool{"host": true, "instance-type": true},
		RemoveHost:   true,
		KeepHostTags: true,
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	pruningConfig = config.ConfigFor("other_app.my_metric")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:       false,
		RemoveTags:   map[string]bool{},
		KeepHostTags: false,
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	pruningConfig = config.ConfigFor("other_app.special")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:       false,
		RemoveTags:   map[string]bool{},
		KeepHostTags: false,
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	pruningConfig = config.ConfigFor("other_app.no_host.my_metric")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:       false,
		RemoveTags:   map[string]bool{"host": true},
		RemoveHost:   true,
		KeepHostTags: false,
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	pruningConfig = config.ConfigFor("other_app.no_host.special")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:       false,
		RemoveTags:   map[string]bool{"host": true},
		RemoveHost:   true,
		KeepHostTags: true,
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	pruningConfig = config.ConfigFor("nothing_fancy")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:       false,
		RemoveTags:   map[string]bool{"host": true},
		RemoveHost:   true,
		KeepHostTags: true,
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	pruningConfig = config.ConfigFor("keep_host")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:       false,
		RemoveTags:   map[string]bool{"whatever": true},
		KeepHostTags: false,
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}
}

// Private helpers

func compareConfigTrees(t *testing.T, expected, actual *configNode, currentPath string) {
	if !reflect.DeepEqual(expected.value, actual.value) {
		t.Errorf("Values differ at path %v: %#v VS %#v", currentPath, expected.value, actual.value)
	}

	for key, expectedChild := range expected.children {
		actualChild := actual.children[key]

		if actualChild == nil {
			t.Errorf("Missing sub-tree at path %v: %#v", normalizePath(currentPath, key), expectedChild)
		} else if !reflect.DeepEqual(expectedChild, actualChild) {
			compareConfigTrees(t, expectedChild, actualChild, normalizePath(currentPath, key))
		}
	}

	for key, actualChild := range actual.children {
		if expected.children[key] == nil {
			t.Errorf("Extra sub-tree at path %v: %#v", normalizePath(currentPath, key), actualChild)
		}
	}
}

func normalizePath(currentPath, key string) string {
	newPath := currentPath
	if newPath != "" {
		newPath += "."
	}
	newPath += key
	return newPath
}
