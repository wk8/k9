package main

import (
	"bytes"
	"io/ioutil"

	"context"
	"io"
	"net/http"
	"strconv"
)

type HttpProxyAction int

const (
	KEEP_AS_IS HttpProxyAction = iota
	TRANSFORM_BODY
	IGNORE_REQUEST
)

type HttpProxyRequestBodyTransformation struct {
	Action HttpProxyAction
	// for any other action than TRANSFORM_BODY, transformedBody should be nil
	TransformedBody io.Reader
}

type HttpProxyRequestBodyTransformer interface {
	Process(*http.Request) (*HttpProxyRequestBodyTransformation, error)
}

type HttpProxy struct {
	Target      string
	Server      *http.Server
	Transformer HttpProxyRequestBodyTransformer
	Client      *http.Client
}

// the target should include the protocol, e.g. http://localhost:8181
// fine for the transformer to be nil
func NewProxy(target string, transformer HttpProxyRequestBodyTransformer) *HttpProxy {
	transport := &http.Transport{
		DisableKeepAlives:   true,
		MaxIdleConnsPerHost: 128,
	}
	client := &http.Client{Transport: transport}

	proxy := &HttpProxy{
		Target:      target,
		Transformer: transformer,
		Client:      client,
	}

	return proxy
}

func (proxy *HttpProxy) Start(localPort int) {
	if proxy.Server != nil {
		logFatal("HttpProxy already started")
	}

	addr := ":" + strconv.Itoa(localPort)
	proxy.Server = &http.Server{Addr: addr, Handler: proxy}

	go func() {
		logInfo("HttpProxy listening on %v", addr)

		if err := proxy.Server.ListenAndServe(); err != nil {
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
	if proxy.Server == nil {
		logFatal("HttpProxy not started yet")
	}

	logInfo("HttpProxy shutting down...")
	proxy.Server.Shutdown(context.Background())
	logInfo("HttpProxy gracefully shut down...")
}

// TODO wkpo refactor en fold?
func (proxy *HttpProxy) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	// transform the body
	transformedBody, err := proxy.transformBody(request)
	if maybeLogErrorAndReply(err, responseWriter, request, "Could not transform body") {
		return
	}
	if transformedBody == nil {
		// we just ignore the request
		logDebug("Ignoring request to %v", request.URL.Path)
		return
	}

	// prepare the request
	clientRequest, err := http.NewRequest(request.Method, proxy.Target+request.URL.Path, transformedBody)
	if maybeLogErrorAndReply(err, responseWriter, request, "Could not create client request") {
		return
	}

	// copy the request headers
	for key, value := range request.Header {
		clientRequest.Header[key] = value
	}

	// make the request downstream
	clientResponse, err := proxy.Client.Do(clientRequest)
	if maybeLogErrorAndReply(err, responseWriter, request, "Unable to make HTTP request downstream") {
		return
	}

	// copy the response headers
	responseHeaders := responseWriter.Header()
	for key, value := range clientResponse.Header {
		responseHeaders[key] = value
	}

	// copy the status code
	responseWriter.WriteHeader(clientResponse.StatusCode)

	// TODO wkpo remove that shit, revert e1dd08cfe01a545ee0549a6d8751cc092b00cd59
	respBodyAsBytes, err := ioutil.ReadAll(clientResponse.Body)
	defer clientResponse.Body.Close()
	if maybeLogErrorAndReply(err, responseWriter, request, "Unable to read HTTP response wkpo") {
		return
	}
	respBody := string(respBodyAsBytes)
	logDebug("wkpo!! %v request for %v:\nresp body: %v\nand headers: %#v", request.Method, request.URL.Path, respBody, clientResponse.Header)
	clientResponse.Body = ioutil.NopCloser(bytes.NewBuffer(respBodyAsBytes))

	// copy the body
	_, err = io.Copy(responseWriter, clientResponse.Body)
	defer clientResponse.Body.Close()
	if maybeLogErrorAndReply(err, responseWriter, request, "Unable to copy response") {
		return
	}
}

func (proxy *HttpProxy) transformBody(request *http.Request) (io.Reader, error) {
	var reader io.Reader

	if proxy.Transformer == nil {
		reader = request.Body
	} else {
		transformation, err := proxy.Transformer.Process(request)
		if err != nil {
			return nil, err
		}

		switch transformation.Action {
		case KEEP_AS_IS:
			reader = request.Body
		case TRANSFORM_BODY:
			reader = transformation.TransformedBody
		case IGNORE_REQUEST:
			reader = nil
		}
	}

	return reader, nil
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
