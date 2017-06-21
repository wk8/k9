package main

import (
	"reflect"
	"testing"
)

func TestNewConfig(t *testing.T) {
	var config *Config

	t.Run("it successfully parses the given config file", func(t *testing.T) {
		output := WithCatpuredLogging(func() {
			config = NewConfig("test_fixtures/configs/all.yml", "")
		})

		if output != "" {
			t.Errorf("Unexpected output: %v", output)
		}

		expectedPruningConfig := NewPruningConfig()
		expectedPruningConfig.MergeWithFile("test_fixtures/pruning_configs/1.yml")
		expectedPruningConfig.MergeWithFile("test_fixtures/pruning_configs/2.yml")

		expectedConfig := &Config{
			PruningConfig: expectedPruningConfig,

			path:        "test_fixtures/configs/all.yml",
			logLevelSet: true,
		}

		if !reflect.DeepEqual(expectedConfig, config) {
			t.Errorf("Unexpected config: %#v", config)
		}
	})
}

func TestReload(t *testing.T) {

}
