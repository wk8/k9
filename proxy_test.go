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
)

type testServer struct{}

var lastRequest *http.Request

func (server *testServer) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	lastRequest = request

	responseWriter.Header()["X-Foo"] = []string{"bar"}

	switch request.URL.Path {
	case "/ping":
		responseWriter.Write([]byte("pong"))
	case "/echo":
		_, err := io.Copy(responseWriter, request.Body)
		if err != nil {
			panic(err)
		}
	default:
		http.Error(responseWriter, "", 404)
	}
}

var transport = &http.Transport{
	DisableKeepAlives: true,
}
var client = &http.Client{Transport: transport}

func TestProxy(t *testing.T) {
	// let's start a simple HTTP server to proxy against
	httpServerPort := getFreePort()
	addr := ":" + strconv.Itoa(httpServerPort)
	httpServer := &http.Server{Addr: addr, Handler: &testServer{}}

	go func() { httpServer.ListenAndServe() }()

	proxyPort := getFreePort()
	proxyTarget := "http://localhost" + addr
	proxyBaseUrl := "http://localhost:" + strconv.Itoa(proxyPort) + "/"
	pingUrl := proxyBaseUrl + "ping"
	echoUrl := proxyBaseUrl + "echo"

	// now to the real tests
	t.Run("it successfully proxies",
		func(t *testing.T) {
			proxy := NewProxy(proxyTarget, nil)
			proxy.Start(proxyPort)

			response, err := http.Get(pingUrl)
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
			proxy := NewProxy(proxyTarget, nil)
			proxy.Start(proxyPort)

			response, err := http.Get(pingUrl)
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
			proxy := NewProxy(proxyTarget, nil)
			proxy.Start(proxyPort)

			request, err := http.NewRequest("GET", pingUrl, nil)
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

	t.Run("when the transformer just passes everything along",
		func(t *testing.T) {
			proxy := NewProxy(proxyTarget, &DummyTransformer{})
			proxy.Start(proxyPort)

			request, err := http.NewRequest("POST", echoUrl, bytes.NewBufferString("hey"))
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

	t.Run("when the transformer says to ignore the request",
		func(t *testing.T) {
			proxy := NewProxy(proxyTarget, &TestTransformer{})
			proxy.Start(proxyPort)

			lastRequest = nil

			request, err := http.NewRequest("POST", echoUrl, bytes.NewBufferString("hey, ignore me!"))
			if err != nil {
				t.Fatal(err)
			}

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}

			// should reply with an empty 200
			if response.StatusCode != 200 {
				t.Errorf("Unexpected status code: %#v", response.StatusCode)
			}
			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != "" {
				t.Errorf("Unexpected body: %#v", string(body))
			}

			// and the server shouldn't have received any request
			if lastRequest != nil {
				t.Errorf("The server did receive a request")
			}

			proxy.Stop()
		})

	t.Run("when the transformer errors out",
		func(t *testing.T) {
			proxy := NewProxy(proxyTarget, &TestTransformer{})
			proxy.Start(proxyPort)

			lastRequest = nil

			request, err := http.NewRequest("POST", echoUrl, bytes.NewBufferString("sadly, error!"))
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
			proxy := NewProxy(proxyTarget, &TestTransformer{})
			proxy.Start(proxyPort)

			request, err := http.NewRequest("POST", echoUrl, bytes.NewBufferString("don't ignore this, but delete me partially"))
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

			// and quite importantly, the server should have received the right
			// content-length
			if !reflect.DeepEqual(lastRequest.Header["Content-Length"], []string{"33"}) {
				t.Errorf("Unexpected header: %#v", lastRequest.Header["Content-Length"])
			}

			proxy.Stop()
		})

	t.Run("when the transformer sends back a longer body",
		func(t *testing.T) {
			proxy := NewProxy(proxyTarget, &TestTransformer{})
			proxy.Start(proxyPort)

			request, err := http.NewRequest("POST", echoUrl, bytes.NewBufferString("if you could double me...?"))
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

			// and quite importantly, the server should have received the right
			// content-length
			if !reflect.DeepEqual(lastRequest.Header["Content-Length"], []string{"35"}) {
				t.Errorf("Unexpected header: %#v", lastRequest.Header["Content-Length"])
			}

			proxy.Stop()
		})

	// now we can stop the server
	httpServer.Shutdown(context.Background())
}

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

// a transformer that just lets everything through
type DummyTransformer struct{}

func (*DummyTransformer) process(*http.Request) (*HttpProxyRequestBodyTransformation, error) {
	return &HttpProxyRequestBodyTransformation{Action: KEEP_AS_IS}, nil
}

// a more complicated transformer:
//  * if the body contains "ignore me", then ignores that request
//  * if the body contains "error!", then returns an error
//  * any occurence of "delete me" in the body is removed, any occurence of
//    "double me" is doubled
//  * if none of the above applies, passes the request through as is
type TestTransformer struct{}

func (*TestTransformer) process(request *http.Request) (*HttpProxyRequestBodyTransformation, error) {
	// read the body
	bodyAsBytes, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		panic(err)
	}
	body := string(bodyAsBytes)
	request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyAsBytes))

	if strings.Contains(body, "ignore me") {
		return &HttpProxyRequestBodyTransformation{Action: IGNORE_REQUEST}, nil
	}
	if strings.Contains(body, "error!") {
		return nil, errors.New("dummy error")
	}

	newBody := strings.Replace(body, "delete me", "", -1)
	newBody = strings.Replace(newBody, "double me", "double medouble me", -1)

	if newBody == body {
		return &HttpProxyRequestBodyTransformation{Action: KEEP_AS_IS}, nil
	} else {
		return &HttpProxyRequestBodyTransformation{Action: TRANSFORM_BODY,
			TransformedBody: strings.NewReader(newBody)}, nil
	}
}
