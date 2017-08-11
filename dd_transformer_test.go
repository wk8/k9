package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
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

	t.Run("for a metric for which it should remove the host and not keep host tags", func(t *testing.T) {
		request := singleMetricRequest(t, "my_app.my_metric")
		if err := transformer.Transform(request); err != nil {
			t.Fatal(err)
		}

		if b := readBody(t, request); b != singleMetricExpectedOutput("my_app.my_metric", false, false) {
			t.Errorf("Unexpected body: %v", b)
		}
	})

	t.Run("for a metric for which it should remove the host and add host tags", func(t *testing.T) {
		request := singleMetricRequest(t, "my_app.special")
		if err := transformer.Transform(request); err != nil {
			t.Fatal(err)
		}

		if b := readBody(t, request); b != singleMetricExpectedOutput("my_app.special", false, true) {
			t.Errorf("Unexpected body: %v", b)
		}
	})

	t.Run("for a metric for which it shouldn't remove the host", func(t *testing.T) {
		request := singleMetricRequest(t, "other_app.my_metric")
		if err := transformer.Transform(request); err != nil {
			t.Fatal(err)
		}

		if b := readBody(t, request); b != singleMetricExpectedOutput("other_app.my_metric", true, false) {
			t.Errorf("Unexpected body: %v", b)
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
	reader := strings.NewReader(singleMetricInput(metricName))
	request, err := http.NewRequest("POST", "http://localhost:8283/api/v1/series/", reader)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func singleMetricInput(metricName string) string {
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

// a tad ugly, but eh less painful than dealing with JSONs again...
func singleMetricExpectedOutput(metricName string, keptHost, hostTagsAdded bool) string {
	format := `{"series":[{"device_name":null,`

	if keptHost {
		format += `"host":"staging-004-e1a",`
	}

	format += `"interval":10,"metric":"%v","points":[[1497975500,104]],"tags":["success:true","timed_out:false","version:87003923341fc1e43469a50bb2e5b6b141210d40","role:my_app"`

	if hostTagsAdded {
		format += `,"security-group:sg-abcd1234","security-group:sg-1234abcd","role:base","role:mysql","tag:aws"`
	}

	format += `],"type":"gauge"}]}`

	return fmt.Sprintf(format, metricName)
}
