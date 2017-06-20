package main

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

// TODO wkpo test on invalid JSON

func TestDDTransformerProcess(t *testing.T) {
	config := NewPruningConfig()
	config.MergeWithFile("test_fixtures/pruning_configs/full.yml")
	transformer := &DDTransformer{config: config}

	t.Run("it doesn't change requests other than POSTs to /api/v1/series/", func(t *testing.T) {
		body := "hey you"

		request, err := http.NewRequest("GET", "http://localhost:8283/api/v1/series/", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		err = transformer.Process(request)
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
		err = transformer.Process(request)
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
		err = transformer.Process(request)
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
		err = transformer.Process(request)
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
			panic(err)
		}

		expectedDecodedBody, err := ioutil.ReadFile("test_fixtures/series_requests/expected_result.json")
		if err != nil {
			t.Fatal(err)
		}

		if b := string(decodedBody); string(expectedDecodedBody) != b {
			t.Errorf("Unexpected body: %v", b)
		}
	})

	t.Run("it cleanly errors out if not fed with a valid JSON", func(t *testing.T) {
		body := "hey you"

		request, err := http.NewRequest("GET", "http://localhost:8283/api/v1/series/", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		err = transformer.Process(request)
		if err != nil {
			// TODO wkpo ca devrait errorer la...
			t.Fatal(err)
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
