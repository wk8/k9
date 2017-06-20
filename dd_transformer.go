package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type DDTransformer struct {
	config *PruningConfig
}

func (transformer *DDTransformer) Process(request *http.Request) error {
	if request.Method == "POST" && request.URL.Path == "/api/v1/series/" {
		return transformer.processSeriesRequest(request)
	}

	return nil
}

func (transformer *DDTransformer) processSeriesRequest(request *http.Request) error {
	reader, encoded, err := maybeDecodeBody(request)
	if err != nil {
		return err
	}

	// parse the JSON
	var jsonDocument map[string]interface{}
	jsonDecoder := json.NewDecoder(reader)
	err = jsonDecoder.Decode(&jsonDocument)
	if err != nil {
		return err
	}

	// transform the body
	transformer.processSeriesRequestJson(jsonDocument)
	newBodyAsBytes, err := json.Marshal(jsonDocument)
	if err != nil {
		return err
	}

	// re-encode if needed
	if encoded {
		newBodyAsBytes = encodeBody(newBodyAsBytes)
	}

	request.Body = ioutil.NopCloser(bytes.NewBuffer(newBodyAsBytes))

	return nil
}

func maybeDecodeBody(request *http.Request) (reader io.ReadCloser, encoded bool, err error) {
	reader = request.Body

	// decode if needed
	contentEncoding := request.Header["Content-Encoding"]
	if len(contentEncoding) > 0 && contentEncoding[0] == "deflate" {
		encoded = true
		reader, err = zlib.NewReader(reader)
	}

	return
}

func (transformer *DDTransformer) processSeriesRequestJson(jsonDocument map[string]interface{}) {
	rawSeries, present := jsonDocument["series"]
	if !present {
		logWarn("Missing the 'series' field in %#v", jsonDocument)
		return
	}
	series, ok := rawSeries.([]interface{})
	if !ok {
		logWarn("'series' not an array %#v", jsonDocument)
		return
	}

	newSeries := []map[string]interface{}{}
	for _, rawMetric := range series {
		metric, ok := rawMetric.(map[string]interface{})
		if !ok {
			logWarn("Unexpected metric in a series JSON (not an object): %#v", rawMetric)
			continue
		}

		name, ok := metric["metric"].(string)
		if !ok {
			logWarn("Unexpected metric in a series JSON (name): %#v", rawMetric)
			continue
		}

		pruningConfig := transformer.config.ConfigFor(name)
		if pruningConfig.Remove {
			continue
		}

		rawTags, present := metric["tags"]
		if present {
			tags, ok := rawTags.([]interface{})
			if !ok {
				logWarn("Unexpected metric in a series JSON (tags): %#v", rawMetric)
				continue
			}

			newTags := []string{}
			for _, rawTag := range tags {
				tag, ok := rawTag.(string)
				if !ok || tag == "" {
					logWarn("Unexpected tag in a series JSON: %#v", rawMetric)
					continue
				}

				splitTag := strings.SplitN(tag, ":", 2)
				if !pruningConfig.RemoveTags[splitTag[0]] {
					newTags = append(newTags, tag)
				}
			}

			if len(newTags) == 0 {
				delete(metric, "tags")
			} else {
				metric["tags"] = newTags
			}
		}

		newSeries = append(newSeries, metric)
	}

	jsonDocument["series"] = newSeries
}

func encodeBody(body []byte) []byte {
	var buffer bytes.Buffer

	writer := zlib.NewWriter(&buffer)
	writer.Write(body)
	writer.Close()

	return buffer.Bytes()
}
