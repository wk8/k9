package main

import (
	"io"
	"log"
	"net/http"
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
	TransformedBody io.ReadCloser
}

type HttpProxyRequestBodyTransformer interface {
	process(*http.Request) (HttpProxyRequestBodyTransformation, error)
}

type HttpProxy struct {
	Transformer *HttpProxyRequestBodyTransformer
	Client      *http.Client
}

// fine for the transformer to be nil
func NewProxy(transformer *HttpProxyRequestBodyTransformer) *HttpProxy {
	transport := &http.Transport{
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 128,
	}

	client := &http.Client{Transport: transport}

	proxy := &HttpProxy{
		Transformer: transformer,
		Client:      client,
	}

	return proxy
}

func (proxy *HttpProxy) Start() {
	http.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		// TODO wkpo pointer to func?
		proxy.HandleRequest(responseWriter, request)
	})

	log.Fatal(http.ListenAndServe(":8081", nil))
}

func (proxy *HttpProxy) HandleRequest(responseWriter http.ResponseWriter, request *http.Request) {
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
	defer transformedBody.Close()

	// prepare the request
	// TODO wkpo config for target...
	clientReq, err := http.NewRequest(request.Method, "http://localhost:8181"+request.URL.Path, transformedBody)
	if maybeLogErrorAndReply(err, responseWriter, request, "Could not create client request") {
		return
	}

	// make the request downstream
	clientResponse, err := proxy.Client.Do(clientReq)
	if maybeLogErrorAndReply(err, responseWriter, request, "Unable to make HTTP request downstream") {
		return
	}

	// copy the headers
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

func (proxy *HttpProxy) transformBody(request *http.Request) (io.ReadCloser, error) {
	var reader io.ReadCloser

	if proxy.Transformer == nil {
		reader = request.Body
	} else {
		transformation, err := (*proxy.Transformer).process(request)
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
		http.Error(responseWriter, "Internal k9 error"+err.Error(), 500)
		return true
	}
}
