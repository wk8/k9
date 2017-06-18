package main

//*/
import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
)

type LoggingTransformer struct{}

func (*LoggingTransformer) Process(request *http.Request) (*HttpProxyRequestBodyTransformation, error) {
	// read the body
	bodyAsBytes, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		panic(err)
	}
	body := string(bodyAsBytes)
	logDebug("wkpo!! %v request for %v:\nbody: %v\nand headers: %#v", request.Method, request.URL.Path, body, request.Header)

	request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyAsBytes))

	return &HttpProxyRequestBodyTransformation{Action: KEEP_AS_IS}, nil
}

func main() {
	// proxy := NewProxy("https://app.datadoghq.com", &LoggingTransformer{})
	proxy := NewProxy("http://localhost:8181", &LoggingTransformer{})
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	proxy.Start(8283)
	<-stop
	proxy.Stop()
}

//*/

/*/
func main() {
	config := NewConfig()
	config.mergeFromFile("test_fixtures/config.yml")
}

//*/
