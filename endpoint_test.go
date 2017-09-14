package main

import (
	"encoding/json"
	"testing"
)

func Test_metricsFromMap(t *testing.T) {
	data := "{\"foo\": 10, \"bar\": 5}"
	var f interface{}
	json.Unmarshal([]byte(data), &f)
	metrics := metricsFromMap(f.(map[string]interface{}), "")
	if len(metrics) != 2 {
		t.Error("wrong number of metrics found")
	}
}
