package main

import (
	"github.com/jamesread/StencilBox/internal/httpserver"
	"github.com/jamesread/StencilBox/internal/buildinfo"
	log "github.com/sirupsen/logrus"
	"os"
)

func setupLogging() {
	if os.Getenv("STENCILBOX_DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	log.WithFields(log.Fields {
		"version":   buildinfo.Version,
		"commit":    buildinfo.Commit,
		"buildDate": buildinfo.BuildDate,
	}).Info("Starting StencilBox")

	setupLogging()

	httpserver.Start()
}
