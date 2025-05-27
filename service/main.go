package main

import (
	"github.com/jamesread/StencilBox/internal/httpserver"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Infof("Starting StencilBox")

	httpserver.Start()
}
