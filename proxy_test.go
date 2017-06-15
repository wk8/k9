package main

import (
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

func TestProxy(t *testing.T) {
	// let's start a simple HTTP server to proxy against
	httpServerPort := getFreePort()
	addr := ":" + strconv.Itoa(httpServerPort)
	httpServer := &http.Server{Addr: addr, Handler: &testServer{}}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

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
				t.Errorf("Unexpected body: %v", string(body))
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

			if response.StatusCode != 200 {
				t.Errorf("Unexpected status code: %v", response.StatusCode)
			}

			response, err = http.Get(proxyBaseUrl + "please_404")
			if err != nil {
				t.Fatal(err)
			}

			if response.StatusCode != 404 {
				t.Errorf("Unexpected status code: %v", response.StatusCode)
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

			response, err := (&http.Client{}).Do(request)
			if err != nil {
				t.Fatal(err)
			}

			// check that the header from the server made it back to the client
			if !reflect.DeepEqual(response.Header["X-Foo"], []string{"bar"}) {
				t.Errorf("Unexpected header: %v", response.Header["X-Foo"])
			}

			// and check that the header from the client made it to the server
			if !reflect.DeepEqual(lastRequest.Header["X-Bar"], []string{"baz"}) {
				t.Errorf("Unexpected header: %v", lastRequest.Header["X-Bar"])
			}

			proxy.Stop()
		})

	t.Run("when the transformer just passes everything along",
		func(t *testing.T) {
			proxy := NewProxy(proxyTarget, &DummyTransformer{})
			proxy.Start(proxyPort)

			response, err := http.Post(echoUrl, "text/html", strings.NewReader("hey"))
			if err != nil {
				t.Fatal(err)
			}

			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			if string(body) != "hey" {
				t.Errorf("Unexpected body: %v", string(body))
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

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// a transformer that just lets everything through
type DummyTransformer struct{}

func (*DummyTransformer) process(*http.Request) (HttpProxyRequestBodyTransformation, error) {
	return HttpProxyRequestBodyTransformation{Action: KEEP_AS_IS}, nil
}

// a more complicated transformer:
//  * if the body contains "ignore me", then ignores that request
//  * if the body contains "error!", then returns an error
//  * any occurence of "delete me" in the body is removed, any occurence of
//    "double me" is doubled
//  * if none of the above applies, passes the request through as is
type TestTransformer struct{}

func (*TestTransformer) process(request *http.Request) (HttpProxyRequestBodyTransformation, error) {
	// read the body
	bodyAsBytes, err := ioutil.ReadAll(request.Body)
	body := string(bodyAsBytes)

	if strings.Contains(body, "ignore me") {
		return HttpProxyRequestBodyTransformation{Action: IGNORE_REQUEST}, nil
	}
	if strings.Contains(body, "error") {
		return nil, errors.New("dummy error")
	}

	newBody := strings.Replace(body, "delete me", "", -1)
	newBody = strings.Replace(newBody, "double me", "double medouble me", -1)

	if newBody == body {
		return HttpProxyRequestBodyTransformation{Action: KEEP_AS_IS}, nil
	} else {
		return HttpProxyRequestBodyTransformation{Action: KEEP_AS_IS,
			TransformedBody: strings.NewReader(newBody)}, nil
	}
}
