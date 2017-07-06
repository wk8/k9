package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type hostTagsTestServer struct{}

var hostTagsTestLastRequest *http.Request
var requestCount int
var nextRequestBody *string
var nextRequestSleep *time.Duration

var interval = 50 * time.Millisecond

func (server *hostTagsTestServer) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	// we simply alternate between 2 JSONs, except if nextRequestBody is not nil
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

	if nextRequestSleep != nil {
		time.Sleep(*nextRequestSleep)
	}

	_, err = responseWriter.Write(responseBody)
	if err != nil {
		panic(err)
	}

	hostTagsTestLastRequest = request
	requestCount++
}

func resetTagsTestServer() {
	// wait for the previous test to be completely done
	time.Sleep(interval)
	hostTagsTestLastRequest = nil
	requestCount = 0
	nextRequestBody = nil
	nextRequestSleep = nil
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

var expectedTags1 = map[string][]string{
	"foo":  []string{"foo:bar"},
	"name": []string{"name:my-aws-host"},
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
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("it retrieves the tags at initialization time", func(t *testing.T) {
		resetTagsTestServer()
		hostTags := NewHostsTags(url, apiKey, applicationKey, nil)

		tags := hostTags.GetTags()
		if !reflect.DeepEqual(expectedTags0, tags) {
			t.Errorf("Unexpected tags: %#v", tags)
		}

		// there should have been exactly one request made to the server
		if requestCount != 1 {
			t.Errorf("Unexpected number of requests: %v", requestCount)
		}

		// and it should be a GET request including the apiKey and the applicationKey
		if hostTagsTestLastRequest.Method != "GET" {
			t.Errorf("Unexpected HTTP method: %v", hostTagsTestLastRequest.Method)
		}
		if hostTagsTestLastRequest.URL.Path != "/api/v1/tags/hosts/"+hostname {
			t.Errorf("Unexpected URL route: %v", hostTagsTestLastRequest.URL.Path)
		}
		if hostTagsTestLastRequest.URL.RawQuery != "api_key=my_awesome_api_key&application_key=my_awesome_application_key" {
			t.Errorf("Unexpected URL query string: %v", hostTagsTestLastRequest.URL.RawQuery)
		}

		// cleanup
		hostTags.Stop()
	})

	t.Run("it periodically updates the tags, and is thread safe", func(t *testing.T) {
		resetTagsTestServer()
		hostTags := NewHostsTags(url, apiKey, applicationKey, &interval)

		// spin up a number of routines, each trying to get tags as fast as they can
		// for 5 intervals
		nbRoutines := 3
		tagsChannel := make(chan []map[string][]string, nbRoutines)
		stopAfter := time.Now().Add(5 * interval)
		for i := 0; i < nbRoutines; i++ {
			go func() {
				localTags := make([]map[string][]string, 0)
				for time.Now().Before(stopAfter) {
					localTags = append(localTags, hostTags.GetTags())
				}
				tagsChannel <- localTags
			}()
		}

		// then wait for each routine to report, and check that each routine saw the
		// expected sequence of tags
		timeout := time.NewTimer(10 * interval)
		allTags := make([][]map[string][]string, 0)
	waiting_loop:
		for i := 0; i < nbRoutines; i++ {
			select {
			case tags := <-tagsChannel:
				allTags = append(allTags, tags)
			case <-timeout.C:
				t.Errorf("Waited for too long, only received tags from %v routine(s)", i)
				break waiting_loop
			}
		}

		for _, tags := range allTags {
			// we should have alternating phases with 0 and 1 tags, and at a couple of phases
			nbPhases := 0
			current := expectedTags1
			next := expectedTags0

			for _, tag := range tags {
				if reflect.DeepEqual(tag, current) {
					continue
				}
				if reflect.DeepEqual(tag, next) {
					nbPhases++
					current, next = next, current
					continue
				}
				t.Errorf("Unexpected tags: %v neither %v nor %v", tag, current, next)
			}

			if nbPhases < 2 {
				t.Errorf("Too few phases: %v", nbPhases)
			}
		}

		// cleanup
		hostTags.Stop()
	})

	t.Run("it times out after a little while if the DD API isn't responsive, but tries to update again later", func(t *testing.T) {
		resetTagsTestServer()
		twoIntervals := 2 * interval
		nextRequestSleep = &twoIntervals
		hostTags := NewHostsTags(url, apiKey, applicationKey, &interval)

		tags := hostTags.GetTags()
		if !reflect.DeepEqual(make(map[string][]string), tags) {
			t.Errorf("Unexpected tags: %#v", tags)
		}
		nextRequestSleep = nil

		// next time should be successful though
		time.Sleep(twoIntervals)
		tags = hostTags.GetTags()
		if !reflect.DeepEqual(expectedTags1, tags) {
			t.Errorf("Unexpected tags: %#v", tags)
		}

		// cleanup
		hostTags.Stop()
	})

	t.Run("if the call fails, but it already has tags, just keeps the current tags", func(t *testing.T) {
		resetTagsTestServer()
		hostTags := NewHostsTags(url, apiKey, applicationKey, &interval)

		twoIntervals := 2 * interval
		nextRequestSleep = &twoIntervals

		tags := hostTags.GetTags()
		if !reflect.DeepEqual(expectedTags0, tags) {
			t.Errorf("Unexpected tags: %#v", tags)
		}

		// cleanup
		hostTags.Stop()
	})

	t.Run("after Stop() is called, it stops updating tags", func(t *testing.T) {
		resetTagsTestServer()
		hostTags := NewHostsTags(url, apiKey, applicationKey, &interval)

		hostTags.Stop()

		time.Sleep(5 * interval)

		// no additional request should have been made
		if requestCount != 1 {
			t.Errorf("Unexpected number of requests: %v", requestCount)
		}
	})

	t.Run("it doesn't crash if the response isn't what's expected", func(t *testing.T) {
		resetTagsTestServer()
		bogusResponseBody := "invalid_response"
		nextRequestBody = &bogusResponseBody
		hostTags := NewHostsTags(url, apiKey, applicationKey, nil)

		tags := hostTags.GetTags()
		if !reflect.DeepEqual(make(map[string][]string), tags) {
			t.Errorf("Unexpected tags: %#v", tags)
		}

		// cleanup
		hostTags.Stop()
	})

	// cleanup
	httpServer.Shutdown(context.Background())
}
