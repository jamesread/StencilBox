package config

import (
	"gopkg.in/yaml.v3"
	"os"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Sites []*Site;
}

type Site struct {
	Name string;
	URL string;
}

func ReadConfigFile() *Config {
	yfile, err := os.ReadFile("config.yaml")

	if err != nil {
		log.Fatal(err)
	}

	cfg := &Config{}

	yaml.Unmarshal(yfile, &cfg)

	return cfg;
}
