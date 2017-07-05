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

func (transformer *DDTransformer) Transform(request *http.Request) error {
	if err := logDebugTransformerRequest(request); err != nil {
		return err
	}

	if request.Method == "POST" && request.URL.Path == "/api/v1/series/" {
		return transformer.transformSeriesRequest(request)
	}

	return nil
}

func (transformer *DDTransformer) transformSeriesRequest(request *http.Request) error {
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
	transformer.transformSeriesRequestJson(jsonDocument)
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

func (transformer *DDTransformer) transformSeriesRequestJson(jsonDocument map[string]interface{}) {
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

		// might seem weird, but the agent does sometimes send a `null` value for tags
		if rawTags, present := metric["tags"]; present && rawTags != nil {
			if rawTags != nil {
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

func logDebugTransformerRequest(request *http.Request) error {
	var err error = nil

	logDebugWith("Received a %v request for %v with body %v", func() []interface{} {
		// hardly super efficient, but then that's not what debug logs are for either
		var reader io.ReadCloser
		var encoded bool
		var bodyAsString string = "<errored when reading body>"
		var bodyAsBytes []byte

		for {
			reader, encoded, err = maybeDecodeBody(request)
			if err != nil {
				break
			}

			bodyAsBytes, err = ioutil.ReadAll(reader)
			defer reader.Close()
			if err != nil {
				break
			}
			bodyAsString = string(bodyAsBytes)

			if encoded {
				bodyAsBytes = encodeBody(bodyAsBytes)
			}
			request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyAsBytes))

			break
		}

		return []interface{}{request.Method, request.URL.Path, bodyAsString}
	})

	return err
}
