package main

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

type RequestTransformer interface {
	Transform(request *http.Request) error
}

type HttpProxy struct {
	target      string
	server      *http.Server
	transformer RequestTransformer
	client      *http.Client
}

// the target should include the protocol, e.g. http://localhost:8181
// it is okay for the transformer to be nil
// the optional timeouts are the connect and global timeouts for requests made
// downstream, respectively defaulting to 5 and 20 secs
func NewProxy(target string, transformer RequestTransformer, optionalTimeouts ...time.Duration) *HttpProxy {
	connectTimeout := 5 * time.Second
	globalTimeout := 20 * time.Second

	switch len(optionalTimeouts) {
	case 2:
		globalTimeout = optionalTimeouts[1]
		fallthrough
	case 1:
		connectTimeout = optionalTimeouts[0]
	case 0:
	default:
		panic("Too many arguments for NewProxy")
	}

	// TODO wkpo next global timeout, and test!!!!
	transport := &http.Transport{
		DisableKeepAlives:   true,
		MaxIdleConnsPerHost: 128,
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, connectTimeout)
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   globalTimeout,
	}

	proxy := &HttpProxy{
		target:      target,
		transformer: transformer,
		client:      client,
	}

	return proxy
}

func (proxy *HttpProxy) Start(localPort int) {
	if proxy.server != nil {
		logFatal("HttpProxy already started")
	}

	addr := ":" + strconv.Itoa(localPort)
	proxy.server = &http.Server{Addr: addr, Handler: proxy}

	go func() {
		logInfo("HttpProxy listening on %v", addr)

		if err := proxy.server.ListenAndServe(); err != nil {
			if err.Error() == "http: Server closed" {
				// normal shutdown
				logInfo("HttpProxy closed")
			} else {
				logFatal("HttpProxy crashed: %#v %T %#v", err.Error(), err, err)
			}
		}
	}()
}

func (proxy *HttpProxy) Stop() {
	if proxy.server == nil {
		logFatal("HttpProxy not started yet")
	}

	logInfo("HttpProxy shutting down...")
	proxy.server.Shutdown(context.Background())
	logInfo("HttpProxy gracefully shut down...")
}

func (proxy *HttpProxy) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	logDebug("Received %v request for %v with headers %#v", request.Method, request.URL.Path, request.Header)

	// transform the request
	if proxy.transformer != nil {
		err := proxy.transformer.Transform(request)
		if maybeLogErrorAndReply(err, responseWriter, request, "Could not transform body") {
			return
		}
	}

	// prepare the request
	pathWithQuery := request.URL.Path
	if len(request.URL.RawQuery) > 0 || request.URL.ForceQuery {
		pathWithQuery += "?" + request.URL.RawQuery
	}
	clientRequest, err := http.NewRequest(request.Method, proxy.target+pathWithQuery, request.Body)
	if maybeLogErrorAndReply(err, responseWriter, request, "Could not create client request") {
		return
	}

	// copy the request headers
	for key, value := range request.Header {
		clientRequest.Header[key] = value
	}

	// make the request downstream
	clientResponse, err := proxy.client.Do(clientRequest)
	if maybeLogErrorAndReply(err, responseWriter, request, "Unable to make HTTP request downstream") {
		return
	}

	logDebugWith("%v request for %v received response with status %v, headers %#v and body %v",
		func() []interface{} {
			// read the request
			respBodyAsBytes, err := ioutil.ReadAll(clientResponse.Body)
			clientResponse.Body = ioutil.NopCloser(bytes.NewBuffer(respBodyAsBytes))
			defer clientResponse.Body.Close()

			var respBody string
			if err == nil {
				respBody = string(respBodyAsBytes)
			} else {
				respBody = "<error reading response body: " + err.Error() + ">"
			}

			return []interface{}{request.Method, request.URL.Path, clientResponse.StatusCode, clientResponse.Header, respBody}
		})

	// copy the response headers
	responseHeaders := responseWriter.Header()
	for key, value := range clientResponse.Header {
		responseHeaders[key] = value
	}

	// copy the status code
	responseWriter.WriteHeader(clientResponse.StatusCode)

	// copy the body
	_, err = io.Copy(responseWriter, clientResponse.Body)
	defer clientResponse.Body.Close()
	if maybeLogErrorAndReply(err, responseWriter, request, "Unable to copy response") {
		return
	}
}

func maybeLogErrorAndReply(err error, responseWriter http.ResponseWriter, request *http.Request, logPrefix string) bool {
	if err == nil {
		return false
	} else {
		logError("%v on path %v: %v", logPrefix, request.URL.Path, err.Error())
		http.Error(responseWriter, "Internal k9 error: "+err.Error(), 500)
		return true
	}
}
