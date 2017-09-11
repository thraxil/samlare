package main // github.com/thraxil/samlare

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/go-kit/kit/log"
)

type endpointconfig struct {
	URL           string
	Prefix        string
	CheckInterval int
	Timeout       int
}

type config struct {
	CarbonHost    string
	CarbonPort    int
	CheckInterval int
	Timeout       int

	Endpoints map[string]endpointconfig
}

func main() {
	configFile := flag.String("config", "/etc/samlare/config.toml", "config file location")
	flag.Parse()

	var logger log.Logger

	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	logger.Log("msg", "starting")

	var conf config
	if _, err := toml.DecodeFile(*configFile, &conf); err != nil {
		logger.Log("msg", "error loading config file", "error", err)
		return
	}

	g := newGraphiteServer(conf.CarbonHost, conf.CarbonPort)

	sigs := make(chan os.Signal, 1)
	for k, endpoint := range conf.Endpoints {
		fmt.Println(k)
		fmt.Println(endpoint.URL)
		elogger := log.With(logger, "endpoint", k)
		e := newEndpoint(endpoint, conf.CheckInterval, conf.Timeout, g, httpFetcher{}, elogger)
		go e.Run()
	}

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	logger.Log("msg", "exiting")
}
