package main // github.com/thraxil/samlare

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/go-kit/kit/log"
)

type renameConfig struct {
	From string
	To   string
}

type endpointconfig struct {
	URL           string
	Prefix        string
	CheckInterval int
	Timeout       int
	FailureMetric string
	IgnoreMetrics []string
	Renames       []renameConfig
}

type config struct {
	CarbonHost    string
	CarbonPort    int
	CheckInterval int
	Timeout       int
	IgnoreMetrics []string
	Renames       []renameConfig

	Endpoint []endpointconfig
}

func loadConfig(configFile string) (*config, error) {
	var conf config
	if _, err := toml.DecodeFile(configFile, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func main() {
	configFile := flag.String("config", "/etc/samlare/config.toml", "config file location")
	flag.Parse()

	var logger log.Logger

	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	logger.Log("msg", "starting")

	conf, err := loadConfig(*configFile)
	if err != nil {
		logger.Log("msg", "error loading config file", "error", err)
		return
	}

	sigs := make(chan os.Signal, 1)

	cancel := startEndpoints(conf, logger)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for {
		s := <-sigs
		if s == syscall.SIGHUP {
			logger.Log("msg", "received SIGHUP. reloading")
			// reload config
			conf, err = loadConfig(*configFile)
			if err != nil {
				// instead of returning, we keep running with the old config
				// logging the fact that there was an error
				logger.Log("msg", "error reloading config file", "error", err)
			} else {
				// kill existing endpoints
				cancel()
				// and restart them with the new config
				cancel = startEndpoints(conf, logger)
			}
		} else {
			logger.Log("msg", "exiting")
			cancel()
			return
		}
	}
}

func startEndpoints(conf *config, logger log.Logger) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	g := newGraphiteServer(conf.CarbonHost, conf.CarbonPort)

	for _, endpoint := range conf.Endpoint {
		elogger := log.With(logger, "endpoint", endpoint.Prefix)
		e := newEndpoint(endpoint, conf.CheckInterval, conf.Timeout, conf.IgnoreMetrics, conf.Renames, g, httpFetcher{}, elogger)
		go e.Run(ctx)
	}

	return cancel
}
