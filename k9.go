package main

// TODO wkpo clean up les imports, et sorter?
import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

// TODO wkpo take the server out to its own file
func main() {
	transport := &http.Transport{
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 128,
	}

	client := &http.Client{Transport: transport}

	http.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		handleRequest(&responseWriter, request, client)
	})

	log.Fatal(http.ListenAndServe(":8081", nil))
}

func handleRequest(responseWriter *http.ResponseWriter, request *http.Request, client *http.Client) {
	// read the body
	body, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if maybeLogErrorAndReply(err, responseWriter, 500, "Could not read body") {
		return
	}

	// TODO wkpo transform it

	// prepare the request
	clientReq, err := http.NewRequest(request.Method, "http://localhost:8181"+request.URL.Path, bytes.NewReader(body))
	if maybeLogErrorAndReply(err, responseWriter, 500, "Could not create client request") {
		return
	}

	// make the request downstream
	clientResponse, err := client.Do(clientReq)
	if maybeLogErrorAndReply(err, responseWriter, 500, "Unable to make HTTP request downstream") {
		return
	}

	// copy the headers
	responseHeaders := (*responseWriter).Header()
	for key, value := range clientResponse.Header {
		responseHeaders[key] = value
	}

	// copy the status code
	(*responseWriter).WriteHeader(clientResponse.StatusCode)

	// copy the body
	_, err = io.Copy(*responseWriter, clientResponse.Body)
	defer clientResponse.Body.Close()
	if maybeLogErrorAndReply(err, responseWriter, 500, "Unable to copy response") {
		return
	}
}

func maybeLogErrorAndReply(err error, responseWriter *http.ResponseWriter, code int, logPrefix string) bool {
	if err == nil {
		return false
	} else {
		logError("%v: %v", logPrefix, err.Error())
		http.Error(*responseWriter, err.Error(), code)
		return true
	}
}
