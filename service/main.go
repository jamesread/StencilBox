package main

import (
	"flag"
	"os"

	"github.com/jamesread/StencilBox/internal/buildinfo"
	"github.com/jamesread/StencilBox/internal/config"
	"github.com/jamesread/StencilBox/internal/httpserver"
	log "github.com/sirupsen/logrus"
)

func setupLogging() {
	if os.Getenv("STENCILBOX_DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	configDir := flag.String("configdir", "", "directory containing config.yaml and optional buildconfigs/")
	flag.Parse()

	if *configDir != "" {
		config.SetConfigDir(*configDir)
	}

	log.WithFields(log.Fields{
		"version":   buildinfo.Version,
		"commit":    buildinfo.Commit,
		"buildDate": buildinfo.BuildDate,
	}).Info("Starting StencilBox")

	setupLogging()

	httpserver.Start()
}
