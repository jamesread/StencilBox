package config

import (
	"gopkg.in/yaml.v3"
	"os"
	log "github.com/sirupsen/logrus"
	"path/filepath"
)

type Config struct {
	OutputDir string
	Sites []*Site;
}

type Site struct {
	Name string;
	URL string;
}

func getConfigFile() string {
	directoriesToSearch := []string{
		"/config",
		"./",
		"../",
	}

	for _, dir := range directoriesToSearch {
		path := filepath.Join(dir, "config.yaml")

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			abs, _ := filepath.Abs(path)
			dir := filepath.Dir(abs)

			log.WithFields(log.Fields{
				"abs": abs,
				"dir": dir,
			}).Infof("Found the config directory")

			return abs
		}
	}

	return "./config.yaml" // Should not exist
}

func ReadConfigFile() *Config {
	yfile, err := os.ReadFile(getConfigFile())

	if err != nil {
		log.Fatal(err)
	}

	cfg := &Config{}

	yaml.Unmarshal(yfile, &cfg)

	return cfg;
}
