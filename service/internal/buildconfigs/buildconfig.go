package buildconfigs

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/jamesread/StencilBox/internal/config"
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

	//  Internal
	Path string

	Datafiles map[string]string

	OnStartup *bool

	Repos []GitRepo

	PostProcessors []string
}

// BuildOnStartup reports whether this config should be built when StencilBox starts.
// Defaults to true when onstartup is omitted from the config file.
func (cfg *BuildConfig) BuildOnStartup() bool {
	if cfg.OnStartup == nil {
		return true
	}
	return *cfg.OnStartup
}

type GitRepo struct {
	URL     string
	Timeout float64
}

func GetConfigDir() (string, error) {
	if dir := config.OverrideDir(); dir != "" {
		return filepath.Join(dir, "buildconfigs"), nil
	}

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

	remoteURL, err := readOriginURL(dir)
	if err != nil {
		return err
	}

	// Use git.CloneOrPull to pull the latest changes
	req := &git.CloneOrPullRequest{
		GitUrl:   remoteURL,
		LocalDir: dir,
		Timeout:  60.0,
		Log:      true,
	}

	logPullResult(git.CloneOrPull(req), remoteURL, dir)

	return nil
}

func logPullResult(res *git.CloneOrPullResult, remoteURL, dir string) {
	if res.WasCloned {
		log.Infof("Cloned repo %s to %s", remoteURL, dir)
		return
	}

	log.Infof("Pulled repo %s in %s", remoteURL, dir)
}

// readOriginURL reads the origin remote URL from the git config in dir.
// It returns os.ErrNotExist when there is no origin URL.
func readOriginURL(dir string) (string, error) {
	configData, err := os.ReadFile(filepath.Join(dir, ".git", "config"))
	if err != nil {
		return "", err
	}

	remoteURL := parseOriginURL(string(configData))
	if remoteURL == "" {
		return "", os.ErrNotExist
	}

	return remoteURL, nil
}

func parseOriginURL(gitConfig string) string {
	lines := strings.Split(gitConfig, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "[remote \"origin\"]") {
			// Look for the url line in the next few lines
			return findURLInSection(lines[i+1 : min(len(lines), i+10)])
		}
	}

	return ""
}

func findURLInSection(lines []string) string {
	for _, line := range lines {
		if url, ok := strings.CutPrefix(line, "\turl = "); ok {
			return url
		}

		if strings.HasPrefix(line, "[") {
			// We've hit the next section
			return ""
		}
	}

	return ""
}

func ReadConfigFiles() map[string]*BuildConfig {
	ret := make(map[string]*BuildConfig, 0)

	dir, err := GetConfigDir()
	if err != nil {
		log.Warnf("No build configs directory found: %v", err)
		return ret
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))

	if files == nil {
		log.Warnf("No config files found in %s", dir)
		return ret
	}

	for _, file := range files {
		if bc := loadBuildConfig(file); bc != nil {
			addUniqueConfig(ret, bc)
		}
	}

	return ret
}

// loadBuildConfig reads a build config file and fills in its internal fields.
func loadBuildConfig(file string) *BuildConfig {
	bc := readBuildConfig(file)

	if bc == nil {
		log.Warnf("Failed to read build config from file: %s", file)
		return nil
	}

	bc.Filename = filepath.Base(file)
	bc.Path = file

	if strings.TrimSpace(bc.Name) == "" {
		bc.Name = strings.TrimSuffix(bc.Filename, filepath.Ext(bc.Filename))
		log.Warnf("Build config %s has no name, using %q", file, bc.Name)
	}

	return bc
}

func addUniqueConfig(configs map[string]*BuildConfig, bc *BuildConfig) {
	if existing, dup := configs[bc.Name]; dup {
		log.Warnf("Build config name %q in %s duplicates %s, renaming to %q", bc.Name, bc.Filename, existing.Filename, bc.Filename)
		bc.Name = bc.Filename
	}

	configs[bc.Name] = bc
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
