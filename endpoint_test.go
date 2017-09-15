package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
)

func Test_metricsFromMap(t *testing.T) {
	data := `{"foo": 10, "bar": 5}`
	var f interface{}
	json.Unmarshal([]byte(data), &f)
	metrics := metricsFromMap(f.(map[string]interface{}), "")
	if len(metrics) != 2 {
		t.Error("wrong number of metrics found")
	}

	data = `{"foo": 10, "bar": 5, "baz": {"blah": 3}}`
	json.Unmarshal([]byte(data), &f)
	metrics = metricsFromMap(f.(map[string]interface{}), "")
	if len(metrics) != 3 {
		t.Error("wrong number of metrics found")
	}

}

func dummyLogger() log.Logger {
	devNull, _ := os.Open("/dev/null")
	w := log.NewSyncWriter(devNull)
	logger := log.NewJSONLogger(w)
	return logger
}

func Test_newEndpoint(t *testing.T) {
	c := endpointconfig{
		URL:           "http://example.com/",
		Prefix:        "test",
		CheckInterval: 60,
		Timeout:       60,
	}
	g := newGraphiteServer("1.2.3.4", 2003)
	e := newEndpoint(c, 60, 60, g, httpFetcher{}, dummyLogger())
	if e.url != c.URL {
		t.Error("lost the URL")
	}
}
