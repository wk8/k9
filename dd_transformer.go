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
	Config *PruningConfig
	// TODO wkpo actually use that shit:
	// 1. add a `keepHostTags` field to remove tags sections, defaulting to true, can be overriden to false, and then overriden back to true
	// 2. in MetricPruningConfig structs, add a `addHostTags` bool, which is there when 'host' is in the list of tags to remove AND keepHostTagsis true for that metric
	// 3. here, when addHostTags is true, then need to fetch them, make them go through the list of tags to remove anyhow
	// 4. update unit tests
	// 5. update the README
	HostTags *HostTags
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
	defer reader.Close()
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
	if isEncodedRequest(request) {
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

		pruningConfig := transformer.Config.ConfigFor(name)
		if pruningConfig.Remove {
			continue
		}

		// remove the host if needed
		// TODO wkpo next unit test on this
		if pruningConfig.RemoveHost {
			delete(metric, "host")
		}

		// now to tags
		newTags := []string{}

		// might seem weird, but the agent does sometimes send a `null` value for tags
		rawTags, present := metric["tags"]
		if present && rawTags != nil {
			if tags, ok := rawTags.([]interface{}); ok {
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
			} else {
				logWarn("Unexpected metric in a series JSON (tags): %#v", rawMetric)
			}
		}

		// host tags, if relevant
		if pruningConfig.KeepHostTags && transformer.HostTags != nil {
			for hostTagName, hostTagValues := range transformer.HostTags.GetTags() {
				if !pruningConfig.RemoveTags[hostTagName] {
					newTags = append(newTags, hostTagValues...)
				}
			}
		}

		if len(newTags) == 0 {
			if rawTags != nil {
				delete(metric, "tags")
			}
		} else {
			metric["tags"] = newTags
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
		var bodyAsBytes []byte
		var bodyAsString string = "<errored when reading body>"

		for {
			bodyAsBytes, err = ioutil.ReadAll(request.Body)
			defer request.Body.Close()
			if err != nil {
				break
			}
			request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyAsBytes))

			decodedBodyAsBytes := bodyAsBytes
			if isEncodedRequest(request) {
				var reader io.ReadCloser
				reader, err = zlib.NewReader(bytes.NewReader(bodyAsBytes))
				if err != nil {
					break
				}

				decodedBodyAsBytes, err = ioutil.ReadAll(reader)
				defer reader.Close()
				if err != nil {
					break
				}
			}

			bodyAsString = string(decodedBodyAsBytes)
			break
		}

		return []interface{}{request.Method, request.URL.Path, bodyAsString}
	})

	return err
}

func isEncodedRequest(request *http.Request) bool {
	contentEncoding := request.Header["Content-Encoding"]
	return len(contentEncoding) > 0 && contentEncoding[0] == "deflate"
}
