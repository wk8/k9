package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestDDTransformerProcess(t *testing.T) {
	config := NewPruningConfig()
	config.MergeWithFileOrGlob("test_fixtures/pruning_configs/full.yml")
	transformer := &DDTransformer{config: config}

	t.Run("it doesn't change requests other than POSTs to /api/v1/series/", func(t *testing.T) {
		body := "hey you"

		request, err := http.NewRequest("GET", "http://localhost:8283/api/v1/series/", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		err = transformer.Transform(request)
		if err != nil {
			t.Fatal(err)
		}

		if b := readBody(t, request); b != body {
			t.Errorf("Unexpected body: %v", b)
		}

		request, err = http.NewRequest("POST", "http://localhost:8283/api/v0/series/", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		err = transformer.Transform(request)
		if err != nil {
			t.Fatal(err)
		}

		if b := readBody(t, request); b != body {
			t.Errorf("Unexpected body: %v", b)
		}
	})

	t.Run("it processes the body according to its pruning configuration", func(t *testing.T) {
		rawContent, err := ioutil.ReadFile("test_fixtures/series_requests/not_encoded.json")
		if err != nil {
			t.Fatal(err)
		}

		request, err := http.NewRequest("POST", "http://localhost:8283/api/v1/series/", bytes.NewReader(rawContent))
		if err != nil {
			t.Fatal(err)
		}
		err = transformer.Transform(request)
		if err != nil {
			t.Fatal(err)
		}

		expectedBody, err := ioutil.ReadFile("test_fixtures/series_requests/expected_result.json")
		if err != nil {
			t.Fatal(err)
		}

		if b := readBody(t, request); b != string(expectedBody) {
			t.Errorf("Unexpected body: %v", b)
		}
	})

	t.Run("it properly decodes and processes encoded requests", func(t *testing.T) {
		rawContent, err := ioutil.ReadFile("test_fixtures/series_requests/encoded")
		if err != nil {
			t.Fatal(err)
		}

		request, err := http.NewRequest("POST", "http://localhost:8283/api/v1/series/", bytes.NewReader(rawContent))
		request.Header["Content-Encoding"] = []string{"deflate"}
		err = transformer.Transform(request)
		if err != nil {
			t.Fatal(err)
		}

		// decode the body
		reader, err := zlib.NewReader(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		decodedBody, err := ioutil.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}

		expectedDecodedBody, err := ioutil.ReadFile("test_fixtures/series_requests/expected_result.json")
		if err != nil {
			t.Fatal(err)
		}

		if b := string(decodedBody); string(expectedDecodedBody) != b {
			t.Errorf("Unexpected body: %v", b)
		}
	})

	t.Run("if debug mode is on, it logs the decoded request", func(t *testing.T) {
		rawContent, err := ioutil.ReadFile("test_fixtures/series_requests/encoded")
		if err != nil {
			t.Fatal(err)
		}

		request, err := http.NewRequest("POST", "http://localhost:8283/api/v1/series/", bytes.NewReader(rawContent))
		request.Header["Content-Encoding"] = []string{"deflate"}

		output := WithLogLevelAndCapturedLogging(DEBUG, func() {
			err = transformer.Transform(request)
			if err != nil {
				t.Fatal(err)
			}
		})

		// check the transformation worked just the same
		reader, err := zlib.NewReader(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		decodedBody, err := ioutil.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		expectedDecodedBody, err := ioutil.ReadFile("test_fixtures/series_requests/expected_result.json")
		if err != nil {
			t.Fatal(err)
		}
		if b := string(decodedBody); string(expectedDecodedBody) != b {
			t.Errorf("Unexpected body: %v", b)
		}

		// and the output should be the decoded request
		if !strings.Contains(output, "my_app.workers.queue_size") {
			t.Errorf("Unexpected logging output: %#v", output)
		}
	})

	t.Run("it cleanly errors out if not fed with a valid JSON", func(t *testing.T) {
		body := "hey you"

		request, err := http.NewRequest("POST", "http://localhost:8283/api/v1/series/", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		err = transformer.Transform(request)
		if err == nil {
			t.Fatal("Didn't get an error")
		}
	})

	t.Run("it doesn't complain about `null` tags, and passes them as is", func(t *testing.T) {
		output := WithCatpuredLogging(func() {
			rawContent, err := ioutil.ReadFile("test_fixtures/series_requests/null_tags.json")
			if err != nil {
				t.Fatal(err)
			}

			request, err := http.NewRequest("POST", "http://localhost:8283/api/v1/series/", bytes.NewReader(rawContent))
			if err != nil {
				t.Fatal(err)
			}
			err = transformer.Transform(request)
			if err != nil {
				t.Fatal(err)
			}

			body := readBody(t, request)

			var initialJson map[string]interface{}
			err = json.Unmarshal(rawContent, &initialJson)
			if err != nil {
				t.Fatal(err)
			}
			var transformedJson map[string]interface{}
			err = json.Unmarshal([]byte(body), &transformedJson)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(initialJson, transformedJson) {
				t.Errorf("Unexpected body: %v", body)
			}
		})

		if output != "" {
			t.Errorf("Unexpected output: %v", output)
		}
	})
}

func TestDDTransformerProcessWithHostTags(t *testing.T) {
	config := NewPruningConfig()
	config.MergeWithFileOrGlob("test_fixtures/pruning_configs/host_tags.yml")
	transformer := NewTransformer(config, &dummyHostTags{})

	t.Run("for a metric for which it should remove the host and not add host tags", func(t *testing.T) {
		request := singleMetricRequest(t, "my_app.my_metric")
		if err := transformer.Transform(request); err != nil {
			t.Fatal(err)
		}

		expectedOutput := singleMetricExpectedOutput(t, "my_app.my_metric", true, []string{})
		if actualOutput := normalizeSeries(parseJson(t, readBody(t, request))); !reflect.DeepEqual(expectedOutput, actualOutput) {
			t.Errorf("Unexpected body:\n%v\nVS expected:\n%v", jsonEncode(t, actualOutput), jsonEncode(t, expectedOutput))
		}
	})

	t.Run("for a metric for which it should remove the host and add host tags", func(t *testing.T) {
		request := singleMetricRequest(t, "my_app.special")
		if err := transformer.Transform(request); err != nil {
			t.Fatal(err)
		}

		// no instance-type in there, since the pruning config specifies to
		// remove that one
		expectedHostTags := []string{"security-group:sg-abcd1234", "security-group:sg-1234abcd", "role:base", "role:mysql", "tag:aws"}
		expectedOutput := singleMetricExpectedOutput(t, "my_app.special", true, expectedHostTags)
		if actualOutput := normalizeSeries(parseJson(t, readBody(t, request))); !reflect.DeepEqual(expectedOutput, actualOutput) {
			t.Errorf("Unexpected body:\n%v\nVS expected:\n%v", jsonEncode(t, actualOutput), jsonEncode(t, expectedOutput))
		}
	})

	t.Run("for a metric for which it shouldn't remove the host", func(t *testing.T) {
		request := singleMetricRequest(t, "other_app.my_metric")
		if err := transformer.Transform(request); err != nil {
			t.Fatal(err)
		}

		expectedOutput := singleMetricExpectedOutput(t, "other_app.my_metric", false, []string{})
		if actualOutput := normalizeSeries(parseJson(t, readBody(t, request))); !reflect.DeepEqual(expectedOutput, actualOutput) {
			t.Errorf("Unexpected body:\n%v\nVS expected:\n%v", jsonEncode(t, actualOutput), jsonEncode(t, expectedOutput))
		}
	})
}

// Private helpers

func readBody(t *testing.T, request *http.Request) string {
	bodyAsBytes, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return string(bodyAsBytes)
}

type dummyHostTags struct{}

func (*dummyHostTags) GetTags() map[string][]string {
	return map[string][]string{
		"instance-type":  []string{"instance-type:m4.large"},
		"security-group": []string{"security-group:sg-abcd1234", "security-group:sg-1234abcd"},
		"role":           []string{"role:base", "role:mysql"},
		"tag":            []string{"tag:aws"},
	}
}

func singleMetricRequest(t *testing.T, metricName string) *http.Request {
	reader := strings.NewReader(singleMetricjsonDocument(metricName))
	request, err := http.NewRequest("POST", "http://localhost:8283/api/v1/series/", reader)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func singleMetricjsonDocument(metricName string) string {
	return fmt.Sprintf(
		`{
           "series": [
             {
               "tags": [
                 "success:true",
                 "timed_out:false",
                 "version:87003923341fc1e43469a50bb2e5b6b141210d40",
                 "role:my_app"
               ],
               "metric": "%v",
               "interval": 10.0,
               "device_name": null,
               "host": "staging-004-e1a",
               "points": [
                 [
                   1497975500.0,
                   104.0
                 ]
               ],
               "type": "gauge"
             }
           ]
         }`, metricName)
}

func singleMetricExpectedOutput(t *testing.T, metricName string, removeHost bool, hostTagsToAdd []string) map[string]interface{} {
	jsonDocument := parseJson(t, singleMetricjsonDocument(metricName))

	series := jsonDocument["series"].([]interface{})
	metric := series[0].(map[string]interface{})

	if removeHost {
		delete(metric, "host")
	}
	metric["tags"] = append(castSliceToStrings(metric["tags"].([]interface{})), hostTagsToAdd...)

	jsonDocument["series"] = []interface{}{metric}

	return normalizeSeries(jsonDocument)
}

func parseJson(t *testing.T, jsonInput string) map[string]interface{} {
	var jsonDocument map[string]interface{}
	if err := json.Unmarshal([]byte(jsonInput), &jsonDocument); err != nil {
		t.Fatal(err)
	}
	return jsonDocument
}

func jsonEncode(t *testing.T, jsonDocument map[string]interface{}) string {
	jsonOutput, err := json.Marshal(jsonDocument)
	if err != nil {
		t.Fatal(err)
	}

	return string(jsonOutput)
}

// because there's no ordering guarantee on how the tags from the host tags
// retriever will be iterated over, we can't just compare JSONs as is: the tags
// ordering might be different than expected
func normalizeSeries(jsonDocument map[string]interface{}) map[string]interface{} {
	series := jsonDocument["series"].([]interface{})

	for _, rawMetric := range series {
		metric := rawMetric.(map[string]interface{})
		rawTags := metric["tags"]
		tags, ok := rawTags.([]string)
		if !ok {
			tags = castSliceToStrings(rawTags.([]interface{}))
		}

		sort.Sort(sort.StringSlice(tags))
		metric["tags"] = tags
	}

	return jsonDocument
}

func castSliceToStrings(slice []interface{}) []string {
	strings := []string{}
	for _, item := range slice {
		strings = append(strings, item.(string))
	}
	return strings
}
