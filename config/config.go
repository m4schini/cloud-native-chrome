package config

import (
	"github.com/JeremyLoy/config"
	"github.com/m4schini/logger"
)

var log = logger.Named("config")
var cfg AppConfig

type AppConfig struct {
	DevMode     bool `config:"DEV"`
	MetricsPort int  `config:"METRICS_PORT"`
	Port        int  `config:"PORT"`
}

func init() {
	err := config.
		FromOptional("app.config").
		FromEnv().
		To(&cfg)
	if err != nil {
		panic(err)
	}

	if cfg.Port == 0 {
		log.Fatal("$PORT is undefined")
	}
}

func DevModeEnabled() bool {
	return cfg.DevMode
}

func MetricsPort() int {
	return cfg.MetricsPort
}

func Port() int {
	return cfg.Port
}
