package main

import (
	"github.com/jamesread/StencilBox/internal/config"
	"github.com/jamesread/StencilBox/internal/httpserver"
//	"github.com/jamesread/StencilBox/internal/generator"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Infof("Starting StencilBox")

	cfg := config.ReadConfigFile()

	httpserver.Start(cfg)
}
