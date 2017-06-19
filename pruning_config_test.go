package main

import (
	"reflect"
	// "strconv"
	"testing"
)

func TestBaseConfig(t *testing.T) {
	config := NewConfig()
	config.MergeWithFile("test_fixtures/configs/full.yml")

	// my_app.elasticsearch.count
	pruningConfig := config.ConfigFor("my_app.elasticsearch.count")
	expectedPruningConfig := &MetricPruningConfig{
		Remove:     false,
		RemoveTags: map[string]bool{"hide_this": true, "role": true, "host": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// my_app.elasticsearch.time.95percentile
	pruningConfig = config.ConfigFor("my_app.elasticsearch.time.95percentile")
	expectedPruningConfig = &MetricPruningConfig{
		Remove:     false,
		RemoveTags: map[string]bool{"hide_this": true, "role": true},
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
		Remove: false,
		// TODO wkpo missing not_for_top_level_metrics?
		RemoveTags: map[string]bool{"hide_this": true, "whatever": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}

	// i_dont_appear_in_the_config
	pruningConfig = config.ConfigFor("i_dont_appear_in_the_config")
	expectedPruningConfig = &MetricPruningConfig{
		Remove: false,
		// TODO wkpo missing not_for_top_level_metrics?
		RemoveTags: map[string]bool{"hide_this": true},
	}
	if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
		t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	}
	//
	//   // my_app.profile.something.avg
	//   pruningConfig = config.ConfigFor("my_app.profile.something.avg")
	//   expectedPruningConfig = &MetricPruningConfig{Remove: true}
	//   if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
	//     t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	//   }
	//
	//   // my_app.profile.something.something.avg
	//   pruningConfig = config.ConfigFor("my_app.profile.something.something.avg")
	//   expectedPruningConfig = &MetricPruningConfig{
	//     Remove:       false,
	//     RemoveTags: map[string]bool{"role": true},
	//   }
	//   if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
	//     t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	//   }
	// }
	//
	// func TestCaching(t *testing.T) {
	//   config := NewConfig()
	//   config.MergeWithFile("test_fixtures/configs/full.yml")
	//
	//   // the cache should be empty
	//   expectedResolvedMetrics := map[string]*configValue{}
	//   if !reflect.DeepEqual(config.resolvedMetrics, expectedResolvedMetrics) {
	//     t.Errorf("Unexpected cache: %#v", config.resolvedMetrics)
	//   }
	//
	//   configValue := config.ConfigFor("my_app.test_caching")
	//   expectedPruningConfig := &MetricPruningConfig{
	//     Remove:       false,
	//     RemoveTags: map[string]bool{"role": true},
	//   }
	//   if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
	//     t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	//   }
	//
	//   // now it should be in the cache
	//   expectedResolvedMetrics["my_app.test_caching"] = expectedPruningConfig
	//   if !reflect.DeepEqual(config.resolvedMetrics, expectedResolvedMetrics) {
	//     t.Errorf("Unexpected cache: %#v", config.resolvedMetrics)
	//   }
	//
	//   // calling a second time should yield the same value
	//   pruningConfig = config.ConfigFor("my_app.test_caching")
	//   if !reflect.DeepEqual(pruningConfig, expectedPruningConfig) {
	//     t.Errorf("Unexpected pruning config: %#v", pruningConfig)
	//   }
}

// func TestSeveralFiles(t *testing.T) {
//   configFromFull := NewConfig()
//   configFromFull.MergeWithFile("test_fixtures/configs/full.yml")
//
//   configFromPartials := NewConfig()
//   for i := 0; i <= 4; i++ {
//     configFromPartials.MergeWithFile("test_fixtures/configs/" + strconv.Itoa(i) + ".yml")
//   }
//
//   if !reflect.DeepEqual(configFromFull, configFromPartials) {
//     t.Errorf("Unexpected configs:\n%#v\nVS\n%#v", configFromFull, configFromPartials)
//   }
// }
