package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

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

type dummyGraphite struct{}

func (d dummyGraphite) Submit(m []metric) error {
	return nil
}

func Test_Submit(t *testing.T) {
	c := endpointconfig{
		URL:           "http://example.com/",
		Prefix:        "test",
		CheckInterval: 60,
		Timeout:       60,
	}
	var g dummyGraphite
	e := newEndpoint(c, 60, 60, g, httpFetcher{}, dummyLogger())
	var m []metric
	err := e.Submit(m)
	if err != nil {
		t.Error("dummy submitter shouldn't return an error")
	}
}

type dummyFetcher struct{}

type dummyReadCloser struct {
	body io.ReadSeeker
}

func (d *dummyReadCloser) Read(p []byte) (n int, err error) {
	n, err = d.body.Read(p)
	if err == io.EOF {
		d.body.Seek(0, 0)
	}
	return n, err
}

func (d *dummyReadCloser) Close() error {
	return nil
}

func NewRespBodyFromString(body string) io.ReadCloser {
	return &dummyReadCloser{strings.NewReader(body)}
}

func (d dummyFetcher) Get(ctx context.Context, url string) (*http.Response, error) {
	data := `{"foo": 10, "bar": 5}`

	r := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       NewRespBodyFromString(data),
		Header:     http.Header{},
	}
	return r, nil
}

func Test_Fetch(t *testing.T) {
	c := endpointconfig{
		URL:           "http://example.com/",
		Prefix:        "test",
		CheckInterval: 60,
		Timeout:       60,
	}
	g := newGraphiteServer("1.2.3.4", 2003)
	e := newEndpoint(c, 60, 60, g, dummyFetcher{}, dummyLogger())

	ctx := context.TODO()

	d, err := e.Fetch(ctx)
	if err != nil {
		t.Error("dummy fetcher shouldn't fail")
	}
	metrics := metricsFromMap(d, "")
	if len(metrics) != 2 {
		t.Error("wrong number of metrics found")
	}

}

func Test_Gather(t *testing.T) {
	c := endpointconfig{
		URL:           "http://example.com/",
		Prefix:        "test",
		CheckInterval: 60,
		Timeout:       60,
	}
	g := newGraphiteServer("1.2.3.4", 2003)
	e := newEndpoint(c, 60, 60, g, dummyFetcher{}, dummyLogger())

	ctx := context.TODO()
	metrics := e.Gather(ctx)

	if len(metrics) != 2 {
		t.Error("wrong number of metrics found")
	}

}

func Test_GatherWithFailureMetric(t *testing.T) {
	c := endpointconfig{
		URL:           "http://example.com/",
		Prefix:        "test",
		CheckInterval: 60,
		Timeout:       60,
		FailureMetric: "failure",
	}
	g := newGraphiteServer("1.2.3.4", 2003)
	e := newEndpoint(c, 60, 60, g, dummyFetcher{}, dummyLogger())

	ctx := context.TODO()
	metrics := e.Gather(ctx)

	if len(metrics) != 3 {
		t.Error("wrong number of metrics found")
	}

}

func Test_Run(t *testing.T) {
	c := endpointconfig{
		URL:           "http://example.com/",
		Prefix:        "test",
		CheckInterval: 60,
		Timeout:       60,
	}
	var g dummyGraphite
	e := newEndpoint(c, 60, 60, g, dummyFetcher{}, dummyLogger())

	ctx, cancel := context.WithCancel(context.TODO())

	// start it and cancel immediately
	go e.Run(ctx)
	cancel()

	// set up a full dummy run
	c = endpointconfig{
		URL:           "http://example.com/",
		Prefix:        "test",
		CheckInterval: 1,
		Timeout:       60,
	}

	e = newEndpoint(c, 60, 60, g, dummyFetcher{}, dummyLogger())

	ctx, cancel = context.WithCancel(context.TODO())

	// start it and cancel immediately
	go e.Run(ctx)

	time.Sleep(2 * time.Millisecond)

	// then cancel it
	cancel()
}
