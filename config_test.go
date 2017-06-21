package main

import (
	"reflect"
	"testing"
)

func TestNewConfig(t *testing.T) {
	var config *Config

	expectedPruningConfig := NewPruningConfig()
	expectedPruningConfig.MergeWithFile("test_fixtures/pruning_configs/1.yml")
	expectedPruningConfig.MergeWithFile("test_fixtures/pruning_configs/2.yml")

	t.Run("it successfully parses the given config file", func(t *testing.T) {
		output := WithCatpuredLogging(func() {
			config = NewConfig("test_fixtures/configs/all.yml", "")
		})

		if output != "" {
			t.Errorf("Unexpected output: %v", output)
		}

		expectedConfig := &Config{
			PruningConfig: expectedPruningConfig,

			path:        "test_fixtures/configs/all.yml",
			logLevelSet: true,
		}

		if !reflect.DeepEqual(expectedConfig, config) {
			t.Errorf("Unexpected config: %#v", config)
		}
	})

	t.Run("it logs warnings if one or more pruning config files can't be parsed", func(t *testing.T) {
		output := WithCatpuredLogging(func() {
			config = NewConfig("test_fixtures/configs/just_pruning_confs.yml", "")
		})

		if !CheckLogLines(t, output, []string{"WARN: Unable to load pruning config from /i/dont/exist: open /i/dont/exist: no such file or directory"}) {
			t.Errorf("Unexpected output: %v", output)
		}

		expectedConfig := &Config{
			PruningConfig: expectedPruningConfig,

			path:        "test_fixtures/configs/just_pruning_confs.yml",
			logLevelSet: false,
		}

		if !reflect.DeepEqual(expectedConfig, config) {
			t.Errorf("Unexpected config: %#v", config)
		}
	})

	t.Run("it crashes with an explicit error if it's unable to build a pruning config", func(t *testing.T) {

	})
}

func TestReload(t *testing.T) {

}
