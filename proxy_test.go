package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type testServer struct{}

var lastRequest *http.Request

func (server *testServer) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	lastRequest = request

	responseWriter.Header()["X-Foo"] = []string{"bar"}

	var err error
	switch request.URL.Path {
	case "/ping":
		_, err = responseWriter.Write([]byte("pong"))
	case "/echo":
		_, err = io.Copy(responseWriter, request.Body)
	case "/echo_qs":
		_, err = responseWriter.Write([]byte(request.URL.RawQuery))
	case "/sleep_25_ms":
		time.Sleep(25 * time.Millisecond)
	default:
		http.Error(responseWriter, "", 404)
	}

	if err != nil {
		panic(err)
	}
}

var transport = &http.Transport{
	DisableKeepAlives: true,
}
var client = &http.Client{Transport: transport}

func TestProxy(t *testing.T) {
	// let's start a simple HTTP server to proxy against
	httpServerPort := getFreePort()
	httpServerPortAsStr := strconv.Itoa(httpServerPort)
	httpServer := &http.Server{Addr: ":" + httpServerPortAsStr, Handler: &testServer{}}
	previousLogLevel := setLogLevel(FATAL)

	go func() { httpServer.ListenAndServe() }()

	proxyTarget := "http://localhost:" + httpServerPortAsStr
	startNewTestProxy := func(transformer RequestTransformer, optionalTimeouts ...time.Duration) (*HttpProxy, string) {
		proxyPort := getFreePort()
		proxyBaseUrl := "http://localhost:" + strconv.Itoa(proxyPort) + "/"

		proxy := NewProxy(proxyTarget, transformer, optionalTimeouts...)
		proxy.Start(proxyPort)

		return proxy, proxyBaseUrl
	}

	// now to the real tests
	t.Run("it successfully proxies",
		func(t *testing.T) {
			proxy, proxyBaseUrl := startNewTestProxy(nil)

			response, err := http.Get(proxyBaseUrl + "ping")
			if err != nil {
				t.Fatal(err)
			}
			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			if string(body) != "pong" {
				t.Errorf("Unexpected body: %#v", string(body))
			}

			proxy.Stop()
		})

	t.Run("it preserves the status code",
		func(t *testing.T) {
			proxy, proxyBaseUrl := startNewTestProxy(nil)

			response, err := http.Get(proxyBaseUrl + "ping")
			if err != nil {
				t.Fatal(err)
			}
			_, err = ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			if response.StatusCode != 200 {
				t.Errorf("Unexpected status code: %#v", response.StatusCode)
			}

			response, err = http.Get(proxyBaseUrl + "please_404")
			if err != nil {
				t.Fatal(err)
			}

			if response.StatusCode != 404 {
				t.Errorf("Unexpected status code: %#v", response.StatusCode)
			}

			proxy.Stop()
		})

	t.Run("it preserves the headers in both directions",
		func(t *testing.T) {
			proxy, proxyBaseUrl := startNewTestProxy(nil)

			request, err := http.NewRequest("GET", proxyBaseUrl+"ping", nil)
			if err != nil {
				t.Fatal(err)
			}
			request.Header["X-Bar"] = []string{"baz"}

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			// check that the header from the server made it back to the client
			if !reflect.DeepEqual(response.Header["X-Foo"], []string{"bar"}) {
				t.Errorf("Unexpected header: %#v", response.Header["X-Foo"])
			}

			// and check that the header from the client made it to the server
			if !reflect.DeepEqual(lastRequest.Header["X-Bar"], []string{"baz"}) {
				t.Errorf("Unexpected header: %#v", lastRequest.Header["X-Bar"])
			}

			proxy.Stop()
		})

	t.Run("it properly copies the query string",
		func(t *testing.T) {
			proxy, proxyBaseUrl := startNewTestProxy(&dummyTransformer{})

			queryString := "hey=you&out=there"

			request, err := http.NewRequest("GET", proxyBaseUrl+"echo_qs?"+queryString, nil)

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			if string(body) != queryString {
				t.Errorf("Unexpected body: %#v", string(body))
			}

			proxy.Stop()
		})

	t.Run("when the transformer does nothing",
		func(t *testing.T) {
			proxy, proxyBaseUrl := startNewTestProxy(&dummyTransformer{})

			request, err := http.NewRequest("POST", proxyBaseUrl+"echo", bytes.NewBufferString("hey"))
			if err != nil {
				t.Fatal(err)
			}

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}

			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			if string(body) != "hey" {
				t.Errorf("Unexpected body: %#v", string(body))
			}

			proxy.Stop()
		})

	t.Run("when the transformer errors out",
		func(t *testing.T) {
			proxy, proxyBaseUrl := startNewTestProxy(&testTransformer{})

			lastRequest = nil

			request, err := http.NewRequest("POST", proxyBaseUrl+"echo", bytes.NewBufferString("sadly, error!"))
			if err != nil {
				t.Fatal(err)
			}

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}

			if response.StatusCode != 500 {
				t.Errorf("Unexpected status code: %#v", response.StatusCode)
			}
			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != "Internal k9 error: dummy error\n" {
				t.Errorf("Unexpected body: %#v", string(body))
			}

			// and the server shouldn't have received any request
			if lastRequest != nil {
				t.Errorf("The server did receive a request")
			}

			proxy.Stop()
		})

	t.Run("when the transformer sends back a shorter body",
		func(t *testing.T) {
			proxy, proxyBaseUrl := startNewTestProxy(&testTransformer{})

			request, err := http.NewRequest("POST", proxyBaseUrl+"echo", bytes.NewBufferString("don't ignore this, but delete me partially"))
			if err != nil {
				t.Fatal(err)
			}

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}

			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != "don't ignore this, but  partially" {
				t.Errorf("Unexpected body: %#v", string(body))
			}

			proxy.Stop()
		})

	t.Run("when the transformer sends back a longer body",
		func(t *testing.T) {
			proxy, proxyBaseUrl := startNewTestProxy(&testTransformer{})

			request, err := http.NewRequest("POST", proxyBaseUrl+"echo", bytes.NewBufferString("if you could double me...?"))
			if err != nil {
				t.Fatal(err)
			}

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}

			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != "if you could double medouble me...?" {
				t.Errorf("Unexpected body: %#v", string(body))
			}

			proxy.Stop()
		})

	t.Run("when the backend fails to connect before the connect timeout expires",
		func(t *testing.T) {
			proxyPort := getFreePort()
			proxy := NewProxy("http://10.255.255.1", nil, 1*time.Millisecond)
			proxy.Start(proxyPort)

			response, err := http.Get("http://localhost:" + strconv.Itoa(proxyPort))
			if err != nil {
				t.Fatal(err)
			}

			if response.StatusCode != 500 {
				t.Errorf("Unexpected status code: %#v", response.StatusCode)
			}

			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			if string(body) != "Internal k9 error: Get http://10.255.255.1/: dial tcp 10.255.255.1:80: i/o timeout\n" {
				t.Errorf("Unexpected body: %#v", string(body))
			}

			proxy.Stop()
		})

	t.Run("when the backend fails to reply before the global timeout expires",
		func(t *testing.T) {
			proxy, proxyBaseUrl := startNewTestProxy(nil, 10*time.Second, 1*time.Millisecond)

			response, err := http.Get(proxyBaseUrl + "sleep_25_ms")
			if err != nil {
				t.Fatal(err)
			}

			if response.StatusCode != 500 {
				t.Errorf("Unexpected status code: %#v", response.StatusCode)
			}

			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			if !strings.HasPrefix(string(body), "Internal k9 error: Get http://localhost:"+httpServerPortAsStr+"/sleep_25_ms: net/http: request canceled") {
				t.Errorf("Unexpected body: %#v", string(body))
			}

			proxy.Stop()
		})

	// now we can stop the server
	httpServer.Shutdown(context.Background())
	// and restore the previous level of logging
	setLogLevel(previousLogLevel)
}

// Private helpers

// asks the kernel for a free open port that is ready to use
func getFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer listen.Close()
	return listen.Addr().(*net.TCPAddr).Port
}

// a transformer that does nothing
type dummyTransformer struct{}

func (*dummyTransformer) Transform(*http.Request) error {
	return nil
}

// a slightly more complicated transformer:
//  * if the body contains "error!", then returns an error
//  * any occurence of "delete me" in the body is removed, any occurence of
//    "double me" is doubled
//  * if none of the above applies, does nothing
type testTransformer struct{}

func (*testTransformer) Transform(request *http.Request) error {
	// read the body
	bodyAsBytes, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		panic(err)
	}
	body := string(bodyAsBytes)

	if strings.Contains(body, "error!") {
		return errors.New("dummy error")
	}

	body = strings.Replace(body, "delete me", "", -1)
	body = strings.Replace(body, "double me", "double medouble me", -1)

	request.Body = ioutil.NopCloser(strings.NewReader(body))
	return nil
}
