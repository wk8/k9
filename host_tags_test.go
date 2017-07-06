package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"testing"
)

type hostTagsTestServer struct{}

var lastRequest *http.Request
var requestCount int
var nextRequestBody *string

func (server *hostTagsTestServer) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	// we simply alternate between 2 JSONs, except if nextRequestBody if not nil
	var jsonName string
	if nextRequestBody != nil {
		jsonName = *nextRequestBody
	} else {
		jsonName = strconv.Itoa(requestCount % 2)
	}
	jsonName = "test_fixtures/host_tags/" + jsonName + ".json"
	responseBody, err := ioutil.ReadFile(jsonName)
	if err != nil {
		panic(err)
	}

	_, err = responseWriter.Write(responseBody)
	if err != nil {
		panic(err)
	}

	lastRequest = request
	requestCount++
}

func resetTagsTestServer() {
	lastRequest = nil
	requestCount = 0
	nextRequestBody = nil
}

var expectedTags0 = map[string][]string{
	"instance-type":     []string{"instance-type:m4.large"},
	"name":              []string{"name:my-aws-host"},
	"region":            []string{"region:us-east-1"},
	"security-group":    []string{"security-group:sg-abcd1234", "security-group:sg-1234abcd"},
	"role":              []string{"role:my_app", "role:base", "role:mysql"},
	"tag":               []string{"tag:aws"},
	"image":             []string{"image:ami-12ab34cd"},
	"env":               []string{"env:production"},
	"availability-zone": []string{"availability-zone:us-east-1a"},
	"kernel":            []string{"kernel:none"},
}

func TestHostTags(t *testing.T) {
	// let's start a simple HTTP server to mock
	httpServerPort := GetFreePort()
	httpServerPortAsStr := strconv.Itoa(httpServerPort)
	httpServer := &http.Server{Addr: ":" + httpServerPortAsStr, Handler: &hostTagsTestServer{}}

	go func() { httpServer.ListenAndServe() }()

	url := "http://localhost:" + httpServerPortAsStr
	apiKey := "my_awesome_api_key"
	applicationKey := "my_awesome_application_key"

	t.Run("it retrieves the tags at initialization time", func(t *testing.T) {
		resetTagsTestServer()
		hostTags := NewHostsTags(url, apiKey, applicationKey, nil)

		tags := hostTags.GetTags()
		if !reflect.DeepEqual(expectedTags0, tags) {
			t.Errorf("Unexpected tags: %#v", tags)
		}

		// and there should have been exactly one request made to the server

		hostTags.Stop()
	})

	// cleanup
	httpServer.Shutdown(context.Background())
}
