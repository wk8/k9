package main

// TODO wkpo clean up les imports, et sorter?
import (
	"errors"
	"fmt"
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
	body, err := ioutil.ReadAll(request.Body)

	if maybeLogErrorAndReply(err, responseWriter, 500, "Could not read body") {
		return
	}

	fmt.Printf("wkpo body %v\n", body)
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
