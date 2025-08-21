package buildconfigs

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/jamesread/golure/pkg/dirs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type BuildConfig struct {
	Name         string
	Filename     string
	OutputDir    string
	ErrorMessage string
	Template     string

	Datafiles map[string]string

	Repos []GitRepo

	PostProcessors []string

	//  Internal
	Path string
}

type GitRepo struct {
	URL string
	Timeout float64
}

func GetConfigDir() (string, error) {
	directoriesToSearch := []string{
		"/config/buildconfigs/",
		"../var/config-skel/buildconfigs/",
		os.Getenv("BUILD_CONFIG_DIR"),
	}

	return dirs.GetFirstExistingDirectory("config", directoriesToSearch)
}

func CanGitPull() bool {
	dir, err := GetConfigDir()

	if err != nil {
		return false
	}

	// Check if the directory has a .git directory
	gitDir := filepath.Join(dir, ".git")
	_, err = os.Stat(gitDir)

	return err == nil
}

func ReadConfigFiles() map[string]*BuildConfig {
	ret := make(map[string]*BuildConfig, 0)

	dir, err := GetConfigDir()

	files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))

	if files == nil {
		log.Warnf("No config files found in %s", dir)
		return ret
	}

	for _, file := range files {
		bc := readBuildConfig(file)
		bc.Filename = filepath.Base(file)
		bc.Path = file

		if bc != nil {
			ret[bc.Name] = bc
		} else {
			log.Warnf("Failed to read build config from file: %s %v", file, err)
		}
	}

	return ret
}

func readBuildConfig(file string) *BuildConfig {
	yfile, err := os.ReadFile(file)

	if err != nil {
		log.Errorf("Error reading BC: %v", err)
		return nil
	}

	cfg := &BuildConfig{}

	decoder := yaml.NewDecoder(bytes.NewReader(yfile))
	decoder.KnownFields(true)
	err = decoder.Decode(&cfg)

	if err != nil {
		log.Errorf("Failed to unmarshal build config from file %s: %v", file, err)
		cfg.ErrorMessage = err.Error()
		return cfg
	}

	return cfg
}
