package main

import (
	"io/ioutil"
	"os"
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
			ListenPort:    8284,
			DdUrl:         "https://my_private.datadoghq.com",

			path:        "test_fixtures/configs/all.yml",
			logLevelSet: true,
		}

		if !reflect.DeepEqual(expectedConfig, config) {
			t.Errorf("Unexpected config: %#v", config)
		}

		if logLevel != DEBUG {
			t.Errorf("Unexpected log level: %v", logLevel)
		}
	})

	t.Run("it logs warnings if one or more pruning config files can't be parsed", func(t *testing.T) {
		output := WithCatpuredLogging(func() {
			config = NewConfig("test_fixtures/configs/just_pruning_confs_1.yml", "")
		})

		if !CheckLogLines(t, output, []string{"WARN: Unable to load pruning config from /i/dont/exist: open /i/dont/exist: no such file or directory"}) {
			t.Errorf("Unexpected output: %v", output)
		}

		expectedConfig := &Config{
			PruningConfig: expectedPruningConfig,

			path:        "test_fixtures/configs/just_pruning_confs_1.yml",
			logLevelSet: false,
		}

		if !reflect.DeepEqual(expectedConfig, config) {
			t.Errorf("Unexpected config: %#v", config)
		}
	})

	t.Run("a log level passed as argument overrides what's in the config file", func(t *testing.T) {
		config = NewConfig("test_fixtures/configs/all.yml", "warn")

		if logLevel != WARN {
			t.Errorf("Unexpected log level: %v", logLevel)
		}
	})
}

func TestNewConfigCrashesWhenFileDoesNotExist(t *testing.T) {
	output := AssertCrashes(t, "TestNewConfigCrashesWhenFileDoesNotExist", func() {
		NewConfig("i/dont/exist", "")
	})

	if !CheckLogLines(t, output, []string{"FATAL: Unable to read the config at i/dont/exist: open i/dont/exist: no such file or directory"}) {
		t.Errorf("Unexpected output: %v", output)
	}
}

// tests that reloading re-parses the pruning config files
func TestReload(t *testing.T) {
	// let's get us a temp file to store configs in
	temp_file, err := ioutil.TempFile("/tmp", "k9-test-reload-config-reload-")
	if err != nil {
		t.Fatal(err)
	}
	temp_path := temp_file.Name()
	err = os.Remove(temp_path)
	if err != nil {
		t.Fatal(err)
	}

	// and let's put the 1st config in it
	err = os.Link("test_fixtures/configs/just_pruning_confs_1.yml", temp_path)
	if err != nil {
		t.Fatal(err)
	}

	// and let's load it!
	config := NewConfig(temp_path, "")

	// now let's keep a pointer to the pruning config
	pruningConfig := config.PruningConfig

	expectedPruningConfig1 := NewPruningConfig()
	expectedPruningConfig1.MergeWithFile("test_fixtures/pruning_configs/1.yml")
	expectedPruningConfig1.MergeWithFile("test_fixtures/pruning_configs/2.yml")

	if !reflect.DeepEqual(expectedPruningConfig1, config.PruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", config.PruningConfig)
	}

	// and let's have the config reload, that shouldn't change anything
	config.Reload()

	if !reflect.DeepEqual(expectedPruningConfig1, config.PruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", config.PruningConfig)
	}
	if config.PruningConfig != pruningConfig {
		t.Errorf("Config pointing to a different pruning config")
	}

	// now let's replace the config with the 2nd one
	err = os.Remove(temp_path)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Link("test_fixtures/configs/just_pruning_confs_2.yml", temp_path)
	if err != nil {
		t.Fatal(err)
	}

	// and reload the config
	config.Reload()

	expectedPruningConfig2 := NewPruningConfig()
	expectedPruningConfig2.MergeWithFile("test_fixtures/pruning_configs/3.yml")
	expectedPruningConfig2.MergeWithFile("test_fixtures/pruning_configs/4.yml")

	if !reflect.DeepEqual(expectedPruningConfig2, config.PruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", config.PruningConfig)
	}
	// but must importantly, the pruning config should be at the same place in
	// memory!
	if config.PruningConfig != pruningConfig {
		t.Errorf("Config pointing to a different pruning config")
	}
}
