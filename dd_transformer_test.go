package main

import (
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

	t.Run("it doesn't change requests other than POSTs to /api/v1/series/",
		func(t *testing.T) {
			body := "hey you"

			request, err := http.NewRequest("GET", "http://localhost:8283/api/v1/series/", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			err = transformer.Process(request)
			if err != nil {
				t.Fatal(err)
			}

			if b := readBody(request); b != body {
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

			if b := readBody(request); b != body {
				t.Errorf("Unexpected body: %v", b)
			}
		})
}

func readBody(request *http.Request) string {
	bodyAsBytes, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		panic(err)
	}
	return string(bodyAsBytes)
}
