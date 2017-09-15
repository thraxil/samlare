package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
)

type fetcher interface {
	Get(context.Context, string) (*http.Response, error)
}

type httpFetcher struct{}

type endpoint struct {
	url            string
	prefix         string
	checkInterval  int
	timeout        int
	graphiteServer Submitable
	fetcher        fetcher
	logger         log.Logger
}

type metric struct {
	Name  string
	Value float64
}

func newEndpoint(c endpointconfig, interval int, timeout int, g Submitable, fetcher fetcher, logger log.Logger) *endpoint {
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

func (h httpFetcher) Get(ctx context.Context, url string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	return http.DefaultClient.Do(req.WithContext(ctx))
}

func (e *endpoint) Fetch(ctx context.Context) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(e.timeout*int(time.Millisecond)))
	defer cancel()
	resp, err := e.fetcher.Get(ctx, e.url)
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

func (e *endpoint) Gather(ctx context.Context) []metric {
	var metrics []metric
	m, err := e.Fetch(ctx)
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

func (e *endpoint) Submit(metrics []metric) error {
	return e.graphiteServer.Submit(metrics)
}

func (e *endpoint) Run(ctx context.Context) {
	e.logger.Log("msg", "endpoint starting")
	for {
		jitter := rand.Intn(e.checkInterval / 10)
		select {
		case <-ctx.Done():
			e.logger.Log("msg", "context cancelled. exiting")
			return
		case <-time.After(time.Duration(e.checkInterval+jitter) * time.Second):
			metrics := e.Gather(ctx)
			err := e.Submit(metrics)
			if err != nil {
				e.logger.Log("msg", "submission to graphite failed", "error", err)
			}
		}
	}
}
