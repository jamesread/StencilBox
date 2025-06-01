package main

import (
	"github.com/jamesread/StencilBox/internal/httpserver"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	log.Infof("Starting StencilBox")

	if os.Getenv("STENCILBOX_DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	httpserver.Start()
}
