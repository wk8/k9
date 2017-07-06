package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// host's tags that we add back to metrics for which we remove the host info
type HostTags struct {
	tags map[string][]string
	// needed when updating the map
	mutex                         sync.RWMutex
	timer                         *time.Timer
	ddUrl, apiKey, applicationKey string
	client                        *http.Client
	timerStopped                  bool
}

const DEFAULT_CACHING_INTERVAL = 1 * time.Hour
const HTTP_TIMEOUT = 20 * time.Second

func NewHostsTags(ddUrl, apiKey, applicationKey string, cachingInterval *time.Duration) *HostTags {
	interval := DEFAULT_CACHING_INTERVAL
	if cachingInterval != nil {
		interval = *cachingInterval
	}

	timeout := HTTP_TIMEOUT
	if interval < timeout {
		timeout = interval
	}
	client := &http.Client{Timeout: timeout}

	hostTags := &HostTags{
		ddUrl:          ddUrl,
		apiKey:         apiKey,
		applicationKey: applicationKey,
		client:         client,
	}
	hostTags.updateTags()
	if hostTags.tags == nil {
		// errored out when initializing
		hostTags.tags = make(map[string][]string)
	}
	started := make(chan bool)
	go hostTags.start(interval, started)
	<-started

	return hostTags
}

func (hostTags *HostTags) GetTags() map[string][]string {
	hostTags.mutex.RLock()
	defer hostTags.mutex.RUnlock()
	return hostTags.tags
}

func (hostTags *HostTags) Stop() {
	hostTags.timer.Stop()
	hostTags.timerStopped = true
}

// Private helpers

func (hostTags *HostTags) start(interval time.Duration, started chan bool) {
	hostTags.timer = time.AfterFunc(interval, func() {
		if hostTags.timerStopped {
			return
		}
		hostTags.updateTags()
		hostTags.timer.Reset(interval)
	})
	started <- true
}

func (hostTags *HostTags) updateTags() {
	newTags, err := hostTags.fetchNewTags()
	if err != nil {
		logError("Unable to retrieve host tags, will be unable to add host tags: %v", err)
		return
	}

	hostTags.mutex.Lock()
	hostTags.tags = newTags
	hostTags.mutex.Unlock()
}

// see https://docs.datadoghq.com/api/?lang=console#tags-get-host
func (hostTags *HostTags) fetchNewTags() (map[string][]string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		logError("Unable to retrieve host name: %v", err)
		return nil, err
	}

	url := fmt.Sprintf("%v/api/v1/tags/hosts/%v?api_key=%v&application_key=%v",
		hostTags.ddUrl, hostname, hostTags.apiKey, hostTags.applicationKey)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logError("Unable to make a request to %v", url)
	}
	response, err := hostTags.client.Do(request)
	if err == nil && response.StatusCode > 299 {
		err = errors.New("status code: " + strconv.Itoa(response.StatusCode))
	}
	if err != nil {
		logError("Unable to retrieve tags for host %v: %v", hostname, err)
		return nil, err
	}

	var jsonDocument map[string]interface{}
	jsonDecoder := json.NewDecoder(response.Body)
	defer response.Body.Close()
	err = jsonDecoder.Decode(&jsonDocument)
	if err != nil {
		return nil, err
	}

	return parseHostTagsResponse(jsonDocument)
}

func parseHostTagsResponse(jsonDocument map[string]interface{}) (map[string][]string, error) {
	rawTags, present := jsonDocument["tags"]
	if !present {
		logError("Missing the 'tags' field in the response from host tags: %#v", jsonDocument)
		return nil, errors.New("malformed response")
	}
	tags, ok := rawTags.([]interface{})
	if !ok {
		logWarn("'tags' not an array %#v", jsonDocument)
		return nil, errors.New("malformed response")
	}

	tagsMap := make(map[string][]string)
	for _, rawTag := range tags {
		tag, ok := rawTag.(string)
		if !ok || tag == "" {
			logWarn("Unexpected tag in the response from host tags: %#v", rawTag)
			continue
		}

		splitTag := strings.SplitN(tag, ":", 2)
		tagsList, present := tagsMap[splitTag[0]]
		if !present {
			tagsList = make([]string, 0, 1)
		}
		tagsMap[splitTag[0]] = append(tagsList, tag)
	}

	return tagsMap, nil
}
