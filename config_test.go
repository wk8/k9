package main

import (
	"reflect"
	"testing"
)

func TestBaseConfig(t *testing.T) {
	config := NewConfig()
	config.mergeFromFile("test_fixtures/configs/blitz.yml")

	// blitz.elasticsearch.count
	configValue := config.ConfigFor("blitz.elasticsearch.count")
	expectedConfigValue := &ConfigValue{
		remove:       false,
		tagsToRemove: map[string]bool{"role": true, "host": true},
	}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// blitz.elasticsearch.time.95percentile
	configValue = config.ConfigFor("blitz.elasticsearch.time.95percentile")
	expectedConfigValue = &ConfigValue{
		remove:       false,
		tagsToRemove: map[string]bool{"role": true},
	}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// blitz.elasticsearch.time.max
	configValue = config.ConfigFor("blitz.elasticsearch.time.max")
	expectedConfigValue = &ConfigValue{remove: true}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// blitz.profile.blitz_partner_order.distribution.time.95percentile
	configValue = config.ConfigFor("blitz.profile.blitz_partner_order.distribution.time.95percentile")
	expectedConfigValue = &ConfigValue{remove: true}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// blitz.profile.something.95percentile
	configValue = config.ConfigFor("blitz.profile.something.95percentile")
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

	// blitz.profile.something.avg
	configValue = config.ConfigFor("blitz.profile.something.avg")
	expectedConfigValue = &ConfigValue{remove: true}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}

	// blitz.profile.something.something.avg
	configValue = config.ConfigFor("blitz.profile.something.something.avg")
	expectedConfigValue = &ConfigValue{
		remove:       false,
		tagsToRemove: map[string]bool{"role": true},
	}
	if !reflect.DeepEqual(configValue, expectedConfigValue) {
		t.Errorf("Unexpected config value: %#v", configValue)
	}
}
