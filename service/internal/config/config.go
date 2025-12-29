package config

import (
	"os"
	"path/filepath"

	"github.com/jamesread/golure/pkg/dirs"
	"github.com/jamesread/httpauthshim/authpublic"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ConfigVersion int                `yaml:"configVersion"`
	Auth          *authpublic.Config `yaml:"auth"`
}

func GetConfigPath() string {
	directoriesToSearch := []string{
		"/config/config.yaml",
		"../var/config-skel/config.yaml",
		"./config.yaml",
		os.Getenv("STENCILBOX_CONFIG_FILE"),
	}

	for _, path := range directoriesToSearch {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Try to find config.yaml in config directory
	configDir, err := dirs.GetFirstExistingDirectory("config", []string{
		"/config/",
		"../var/config-skel/",
		"./",
	})
	if err == nil {
		configPath := filepath.Join(configDir, "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return ""
}

func LoadConfig() *Config {
	cfg := &Config{
		ConfigVersion: 1,
	}

	configPath := GetConfigPath()
	if configPath == "" {
		log.Debug("No config.yaml file found, using defaults")
		return cfg
	}

	log.Infof("Loading config from: %s", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Warnf("Failed to read config file %s: %v", configPath, err)
		return cfg
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		log.Warnf("Failed to parse config file %s: %v", configPath, err)
		return cfg
	}

	log.Debugf("Config loaded: %+v", cfg)
	return cfg
}
