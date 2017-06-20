package main

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
	"net/http"
)

type DDTransformer struct {
	reqBody string
}

func (transformer *DDTransformer) Process(request *http.Request) (*HttpProxyRequestBodyTransformation, error) {
	if err := transformer.readBody(request); err != nil {
		return nil, err
	}

	logDebug("Body for %v: %v", request.URL.Path, transformer.reqBody)

	return &HttpProxyRequestBodyTransformation{Action: KEEP_AS_IS}, nil
}

func (transformer *DDTransformer) readBody(request *http.Request) error {
	// read the body
	bodyAsBytes, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		return err
	}

	// TODO wkpo meme pas besoin de ca si?
	request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyAsBytes))

	contentEncoding := request.Header["Content-Encoding"]
	if len(contentEncoding) > 0 && contentEncoding[0] == "deflate" {
		zlibReader, err := zlib.NewReader(bytes.NewBuffer(bodyAsBytes))
		if err != nil {
			return err
		}

		bodyAsBytes, err = ioutil.ReadAll(zlibReader)
		defer zlibReader.Close()
		if err != nil {
			return err
		}
	}

	transformer.reqBody = string(bodyAsBytes)
	return nil
}
