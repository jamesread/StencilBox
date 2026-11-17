package config

import (
	"os"
	"path/filepath"

	"github.com/jamesread/golure/pkg/dirs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// Auth is the httpauthshim configuration (JWT, trusted headers, mTLS), passed to
	// armature-iam as-is. See https://github.com/jamesread/httpauthshim
	Auth map[string]any `yaml:"auth"`

	Database DatabaseConfig `yaml:"database"`

	ConfigVersion int `yaml:"configVersion"`
}

// DatabaseConfig controls the SQLite database that holds users, groups, roles and API keys.
type DatabaseConfig struct {
	// Enabled defaults to true. Without a database, StencilBox only accepts requests from
	// localhost, and user accounts, IAM and API keys are unavailable.
	Enabled *bool `yaml:"enabled"`
}

// DatabaseEnabled reports whether to use the database: config.yaml database.enabled, unless
// STENCILBOX_DB_DISABLED=true turns it off.
func (c *Config) DatabaseEnabled() bool {
	if os.Getenv("STENCILBOX_DB_DISABLED") == "true" {
		return false
	}
	return c.Database.Enabled == nil || *c.Database.Enabled
}

// overrideDir is set via -configdir and takes priority over search paths.
var overrideDir string

func SetConfigDir(dir string) {
	overrideDir = dir
}

func OverrideDir() string {
	return overrideDir
}

func GetConfigPath() string {
	if overrideDir != "" {
		return filepath.Join(overrideDir, "config.yaml")
	}

	directoriesToSearch := []string{
		"/config/config.yaml",
		"../var/config-skel/config.yaml",
		"./config.yaml",
		os.Getenv("STENCILBOX_CONFIG_FILE"),
	}

	if path := firstExistingFile(directoriesToSearch); path != "" {
		return path
	}

	return configPathInConfigDir()
}

func firstExistingFile(paths []string) string {
	for _, path := range paths {
		if path != "" && fileExists(path) {
			return path
		}
	}

	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// configPathInConfigDir tries to find config.yaml in the config directory.
func configPathInConfigDir() string {
	configDir, err := dirs.GetFirstExistingDirectory("config", []string{
		"/config/",
		"../var/config-skel/",
		"./",
	})
	if err != nil {
		return ""
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if fileExists(configPath) {
		return configPath
	}

	return ""
}

// ConfigDir is the directory holding config.yaml, which is also where the database lives.
func ConfigDir() string {
	if overrideDir != "" {
		return overrideDir
	}
	if p := GetConfigPath(); p != "" {
		return filepath.Dir(p)
	}
	if info, err := os.Stat("/config"); err == nil && info.IsDir() {
		return "/config"
	}
	return "."
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

	if _, ok := cfg.Auth["localUsers"]; ok {
		log.Warn("auth.localUsers in config.yaml is no longer used: users now live in the StencilBox database. Log in as admin and create them under Control Panel > IAM.")
	}

	log.Debugf("Config loaded from %s", configPath)
	return cfg
}
