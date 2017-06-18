package main

import (
	"reflect"
	"strconv"
	"testing"
)

func TestBaseConfig(t *testing.T) {
	config := NewConfig()
	config.MergeFromFile("test_fixtures/configs/full.yml")

	// my_proj.elasticsearch.count
	configValue := config.ConfigFor("my_proj.elasticsearch.count")
	expectedConfigValue := &ConfigValue{
		remove:       false,
		tagsToRemove: map[string]bool{"role": true, "host": true},
	}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// my_proj.elasticsearch.time.95percentile
	configValue = config.ConfigFor("my_proj.elasticsearch.time.95percentile")
	expectedConfigValue = &ConfigValue{
		remove:       false,
		tagsToRemove: map[string]bool{"role": true},
	}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// my_proj.elasticsearch.time.max
	configValue = config.ConfigFor("my_proj.elasticsearch.time.max")
	expectedConfigValue = &ConfigValue{remove: true}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// my_proj.elasticsearch.time.min
	configValue = config.ConfigFor("my_proj.elasticsearch.time.min")
	expectedConfigValue = &ConfigValue{remove: true}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// my_proj.profile.my_proj_partner_order.distribution.time.95percentile
	configValue = config.ConfigFor("my_proj.profile.my_proj_partner_order.distribution.time.95percentile")
	expectedConfigValue = &ConfigValue{remove: true}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// my_proj.profile.something.95percentile
	configValue = config.ConfigFor("my_proj.profile.something.95percentile")
	expectedConfigValue = &ConfigValue{remove: true}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// top_level_metric
	configValue = config.ConfigFor("top_level_metric")
	expectedConfigValue = &ConfigValue{remove: true}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// another_top_level_metric
	configValue = config.ConfigFor("another_top_level_metric")
	expectedConfigValue = &ConfigValue{
		remove:       false,
		tagsToRemove: map[string]bool{"whatever": true},
	}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// i_dont_appear_in_the_config
	configValue = config.ConfigFor("i_dont_appear_in_the_config")
	expectedConfigValue = &ConfigValue{
		remove:       false,
		tagsToRemove: map[string]bool{},
	}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// my_proj.profile.something.avg
	configValue = config.ConfigFor("my_proj.profile.something.avg")
	expectedConfigValue = &ConfigValue{remove: true}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// my_proj.profile.something.something.avg
	configValue = config.ConfigFor("my_proj.profile.something.something.avg")
	expectedConfigValue = &ConfigValue{
		remove:       false,
		tagsToRemove: map[string]bool{"role": true},
	}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}
}

func TestCaching(t *testing.T) {
	config := NewConfig()
	config.MergeFromFile("test_fixtures/configs/full.yml")

	// the cache should be empty
	expectedResolvedMetrics := map[string]*ConfigValue{}
	if !reflect.DeepEqual(config.resolvedMetrics, expectedResolvedMetrics) {
		t.Errorf("Unexpected cache: %#v", config.resolvedMetrics)
	}

	configValue := config.ConfigFor("my_proj.test_caching")
	expectedConfigValue := &ConfigValue{
		remove:       false,
		tagsToRemove: map[string]bool{"role": true},
	}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// now it should be in the cache
	expectedResolvedMetrics["my_proj.test_caching"] = expectedConfigValue
	if !reflect.DeepEqual(config.resolvedMetrics, expectedResolvedMetrics) {
		t.Errorf("Unexpected cache: %#v", config.resolvedMetrics)
	}

	// calling a second time should yield the same value
	configValue = config.ConfigFor("my_proj.test_caching")
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}
}

func TestSeveralFiles(t *testing.T) {
	configFromFull := NewConfig()
	configFromFull.MergeFromFile("test_fixtures/configs/full.yml")

	configFromPartials := NewConfig()
	for i := 0; i <= 4; i++ {
		configFromPartials.MergeFromFile("test_fixtures/configs/" + strconv.Itoa(i) + ".yml")
	}

	if !reflect.DeepEqual(configFromFull, configFromPartials) {
		t.Errorf("Unexpected configs:\n%#v\nVS\n%#v", configFromFull, configFromPartials)
	}
}
