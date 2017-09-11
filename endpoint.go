package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
)

type fetcher interface {
	Get(string, time.Duration) (*http.Response, error)
}

type httpFetcher struct{}

type endpoint struct {
	url            string
	prefix         string
	checkInterval  int
	timeout        int
	graphiteServer *graphiteServer
	fetcher        fetcher
	logger         log.Logger
}

type metric struct {
	Name  string
	Value float64
}

func newEndpoint(c endpointconfig, interval int, timeout int, g *graphiteServer, fetcher fetcher, logger log.Logger) *endpoint {
	if c.CheckInterval != 0 {
		interval = c.CheckInterval
	}
	if c.Timeout != 0 {
		timeout = c.Timeout
	}

	return &endpoint{
		url:            c.URL,
		prefix:         c.Prefix,
		checkInterval:  interval,
		timeout:        timeout,
		graphiteServer: g,
		fetcher:        fetcher,
		logger:         logger,
	}
}

func (h httpFetcher) Get(url string, timeout time.Duration) (*http.Response, error) {
	client := http.Client{
		Timeout: timeout,
	}
	return client.Get(url)
}

func (e *endpoint) Fetch() (map[string]interface{}, error) {
	timeout := time.Duration(e.timeout * int(time.Millisecond))
	resp, err := e.fetcher.Get(e.url, timeout)
	if err != nil {
		return nil, err
	}
	if resp.Status != "200 OK" {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	var f interface{}
	err = json.Unmarshal(b, &f)
	if err != nil {
		return nil, err
	}
	return f.(map[string]interface{}), nil
}

func (e *endpoint) Gather() []metric {
	var metrics []metric
	m, err := e.Fetch()
	if err != nil {
		e.logger.Log("msg", "fetch failed", "error", err)
		return metrics
	}
	e.logger.Log("msg", "good fetch")

	metrics = metricsFromMap(m, e.prefix)

	return metrics
}

func metricsFromMap(m map[string]interface{}, prefix string) []metric {
	var metrics []metric
	for k, v := range m {
		key := fmt.Sprintf("%s.%s", prefix, k)
		switch vv := v.(type) {
		case int:
			metrics = append(metrics, metric{key, float64(vv)})
		case float64:
			metrics = append(metrics, metric{key, vv})
		case map[string]interface{}:
			nmetrics := metricsFromMap(vv, key)
			for _, met := range nmetrics {
				metrics = append(metrics, met)
			}
		}
		// default: nothing to do
	}
	return metrics
}

func (e *endpoint) Submit(metrics []metric) {
	err := e.graphiteServer.Submit(metrics)
	if err != nil {
		e.logger.Log("msg", "submission to graphite failed", "error", err)
	}
}

func (e *endpoint) Run() {
	for {
		metrics := e.Gather()
		e.Submit(metrics)
		jitter := rand.Intn(e.checkInterval / 10)
		time.Sleep(time.Duration(e.checkInterval+jitter) * time.Second)
	}
}
