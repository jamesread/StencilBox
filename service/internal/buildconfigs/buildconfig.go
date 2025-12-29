package buildconfigs

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/jamesread/golure/pkg/dirs"
	"github.com/jamesread/golure/pkg/git"
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

// GitPull performs a git pull on the build configurations directory
func GitPull() error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Check if it's a git repository
	if !CanGitPull() {
		return os.ErrNotExist
	}

	// Get the remote URL from git config
	gitConfigPath := filepath.Join(dir, ".git", "config")
	configData, err := os.ReadFile(gitConfigPath)
	if err != nil {
		return err
	}

	// Parse the config to find the remote URL
	remoteURL := ""
	lines := strings.Split(string(configData), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "[remote \"origin\"]") {
			// Look for the url line in the next few lines
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				if strings.HasPrefix(lines[j], "\turl = ") {
					remoteURL = strings.TrimPrefix(lines[j], "\turl = ")
					break
				}
				if strings.HasPrefix(lines[j], "[") {
					// We've hit the next section
					break
				}
			}
			break
		}
	}

	if remoteURL == "" {
		return os.ErrNotExist
	}

	// Use git.CloneOrPull to pull the latest changes
	req := &git.CloneOrPullRequest{
		GitUrl:   remoteURL,
		LocalDir: dir,
		Timeout:  60.0,
		Log:      true,
	}

	res := git.CloneOrPull(req)
	if res.WasCloned {
		log.Infof("Cloned repo %s to %s", remoteURL, dir)
	} else {
		log.Infof("Pulled repo %s in %s", remoteURL, dir)
	}

	return nil
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
