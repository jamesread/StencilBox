package buildconfigs

import (
	"gopkg.in/yaml.v3"
	"os"
	log "github.com/sirupsen/logrus"
	"github.com/jamesread/golure/pkg/dirs"
	"path/filepath"
)

type BuildConfig struct {
	Name string

	OutputDir string

	Template string

	Datafiles map[string]string

	//  Internal
	Path string
}

func getConfigDir() (string, error) {
	directoriesToSearch := []string{
		"/config/buildconfigs/",
		"../var/config-skel/buildconfigs/",
		os.Getenv("BUILD_CONFIG_DIR"),
	}

	return dirs.GetFirstExistingDirectory("config", directoriesToSearch)
}

func ReadConfigFiles() map[string]*BuildConfig {
	ret := make(map[string]*BuildConfig, 0)

	dir, err := getConfigDir()

	files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))

	if files == nil {
		log.Warnf("No config files found in %s", dir)
		return ret
	}

	for _, file := range files {
		bc := readBuildConfig(file)
		bc.Path = file

		if bc != nil {
			ret[bc.Name] = bc
		} else {
			log.Warnf("Failed to read build config from file: %s %v", file, err)
		}
	}

	return ret;
}

func readBuildConfig(file string) *BuildConfig {
	yfile, err := os.ReadFile(file)

	if err != nil {
		log.Errorf("Error reading BC: %v", err)
		return nil
	}

	cfg := &BuildConfig{}

	err = yaml.Unmarshal(yfile, &cfg)

	if err != nil {
		log.Errorf("Failed to unmarshal build config from file %s: %v", file, err)
		return nil
	}

	return cfg
}
