package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
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
			t.Fatal(nil)
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
			t.Fatal(nil)
		}

		// decode the body
		reader, err := zlib.NewReader(request.Body)
		if err != nil {
			t.Fatal(nil)
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
				t.Fatal(nil)
			}
		})

		// check the transformation worked just the same
		reader, err := zlib.NewReader(request.Body)
		if err != nil {
			t.Fatal(nil)
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
				t.Fatal(nil)
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

		logInfo("wkpo bordel output: %v", output)
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
