package generator

import (
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/golure/pkg/dirs"
	"github.com/jamesread/golure/pkg/git"
	"github.com/jamesread/golure/pkg/easyexec"
	"errors"

	"fmt"
	"math"
	"os"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"

	"path/filepath"

	"text/template"
)

type BuildStatus struct {
	IsError bool
	Message string
	BuildUrlBase string
	OutputSizeHumanReadable string
}

func Generate(baseOutputDir string, cfg *buildconfigs.BuildConfig, buildStatus *BuildStatus, updateChannel chan string) {
	updateChannel <- fmt.Sprintf("Starting build for project %s", cfg.Name)

    defer close(updateChannel)

	finalOutputDir := filepath.Join(baseOutputDir, cfg.OutputDir)
	temporaryOutputDir := filepath.Join(baseOutputDir, cfg.OutputDir + "_tmp")

	log.WithFields(log.Fields{
		"name":	   cfg.Name,
		"outputDir": cfg.OutputDir,
		"finalOutputDir": finalOutputDir,
	}).Infof("Starting build for project")

	os.MkdirAll(finalOutputDir, 0755)
	os.MkdirAll(temporaryOutputDir, 0755)

	indexPath := filepath.Join(FindTemplateDir(), cfg.Template, "index.html")

	tmpl, err := template.ParseFiles(indexPath)

	if err != nil {
		updateChannel <- "Failed to parse template: " + err.Error()

		buildStatus.IsError = true
		buildStatus.Message = "Failed to parse template: " + err.Error()
        return
	}

	templateData := make(map[string]any)

	for name, path := range cfg.Datafiles {
		log.Infof("Adding datafile %v with path %v", name, path)

		dat, err := readDatafile(path, filepath.Dir(cfg.Path))

		if err != nil {
			buildStatus.IsError = true
			buildStatus.Message = "Error reading datafile " + name + ": " + err.Error()
			return
		}

		if dat == nil {
			continue
		}

		templateData[name] = dat
	}

	log.Infof("Datafiles loaded: %+v", templateData)

	outfile, err := os.Create(temporaryOutputDir + "/index.html")

	if err != nil {
		log.Errorf("Error creating output file: %v", err)
		return
	}

	repoStorageDir := getRepoStorageDir(cfg)
	cloneRepos(cfg.Repos, repoStorageDir, updateChannel)

	templateData["hooks"] = buildRepoHooks(cfg, updateChannel)
	templateData["buildconfig"] = cfg

	err = tmpl.Execute(outfile, templateData)

	if err != nil {
		buildStatus.IsError = true
		buildStatus.Message = "Error executing template: " + err.Error()
		return
	}

	copyLayers(temporaryOutputDir)

	lnRepos(temporaryOutputDir, repoStorageDir)

	updateChannel <- "Running Vite build..."

	runVite(temporaryOutputDir, finalOutputDir, cfg.OutputDir)

	updateChannel <- "Build completed!"

	buildStatus.Message = "Build completed successfully"
	buildStatus.IsError = false
	buildStatus.BuildUrlBase = getBuildUrlBase()
	buildStatus.OutputSizeHumanReadable = getDirectorySizeHumanReadable(finalOutputDir)

	return
}

func buildRepoHooks(cfg *buildconfigs.BuildConfig, updateChannel chan string) *map[string]string {
	updateChannel <- "Building repo hooks..."

	hooks := make(map[string]string)

	for _, repo := range cfg.Repos {
		repoName := filepath.Base(repo.URL)
		repoName = repoName[:len(repoName)-len(filepath.Ext(repoName))]

		appendHookData("head", &hooks, &repo, cfg)
		appendHookData("body", &hooks, &repo, cfg)
	}

	return &hooks
}

func appendHookData(hookName string, hooks *map[string]string, repo *buildconfigs.GitRepo, cfg *buildconfigs.BuildConfig) {
	if _, ok := (*hooks)[hookName]; ok {
		(*hooks)[hookName] = ""
	}

	(*hooks)[hookName] += getHookData(hookName, repo, cfg) + "\n"
}

func getHookData(hookName string, repo *buildconfigs.GitRepo, cfg *buildconfigs.BuildConfig) string {
	log.Infof("Getting hook data for %v, repo: %s", hookName, repo.URL)

	repoPath := filepath.Join(getRepoStorageDir(cfg), filepath.Base(repo.URL))

	htmlFilename := filepath.Join(repoPath, hookName + ".html")

	if _, err := os.Stat(htmlFilename); errors.Is(err, os.ErrNotExist) {
		log.Infof("No %s hook file found for repo %s", hookName, repo.URL)
		return ""
	}

	contents, err := os.ReadFile(htmlFilename)

	if err != nil {
		log.Errorf("Error reading %s hook file for repo %s: %v", hookName, repo.URL, err)
		return ""
	}

	return string(contents)
}

func getRepoStorageDir(cfg *buildconfigs.BuildConfig) string {
	repoStorageDir := filepath.Join(filepath.Dir(cfg.Path), "repos")

	// Check if the directory exists
	if _, err := os.Stat(repoStorageDir); os.IsNotExist(err) {
		os.MkdirAll(repoStorageDir, 0755)
	}

	log.Infof("repoPath:" + repoStorageDir)

	return repoStorageDir
}

func lnRepos(finalOutputDir string, repoStorageDir string) {
	repoLinkPath := filepath.Join(finalOutputDir, "repos")

	err := os.Symlink(repoStorageDir, repoLinkPath)

	if err != nil {
		log.Errorf("Error creating symlink to repo storage directory: %v", err)
		return
	}
}

func runVite(temporaryOutputDir string, finalOutputDir string, base string) {
	req := &easyexec.ExecRequest{
		Executable: "vite",
		Args: []string{"build", "--outDir", finalOutputDir, "--base", "/" + base + "/"},
		WorkingDirectory: temporaryOutputDir,
	}

	res := easyexec.ExecWithRequest(req)

	log.Infof("Vite build completed with exit code %d", res.ExitCode)
}

func getDirectorySizeHumanReadable(dir string) string {
	var size int64 = 0

	err := filepath.Walk(dir, func(_ string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	if err != nil {
		log.Errorf("Error calculating directory size: %v", err)
		return "unknown"
	}

	return formatSize(size)
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
	}
}

func getBuildUrlBase() string {
	if os.Getenv("STENCILBOX_BUILD_URL_BASE") != "" {
		return os.Getenv("STENCILBOX_BUILD_URL_BASE")
	}

	return ""
}

func cloneRepos(repos []buildconfigs.GitRepo, outputDir string, updateChannel chan string) {
	for _, repo := range repos {
		updateChannel <- fmt.Sprintf("Cloning repo %s", repo.URL)

        cloneRepo(repo, outputDir)
	}
}

func cloneRepo(repo buildconfigs.GitRepo, outputDir string) {
	repo.Timeout = math.Max(repo.Timeout, 60.0)

	log.Infof("Cloning or pulling repo %s with timeout %f seconds", repo.URL, repo.Timeout)

	req := &git.CloneOrPullRequest{
		GitUrl: repo.URL,
		LocalDir: outputDir,
		Timeout: repo.Timeout,
		Log: true,
	}

	res := git.CloneOrPull(req)

	if res.WasCloned {
		log.Infof("Cloned repo %s to %s", repo.URL, outputDir)
	} else {
		log.Infof("Pulled repo %s in %s", repo.URL, outputDir)
	}
}

func findLayersDir() string {
	dir, _ := dirs.GetFirstExistingDirectory("layers", []string{
		"../layers/",
		"/app/layers/",
		"/config/layers/",
	})

	if dir == "" {
		log.Warnf("No layers directory found, using default")
		return "/config/layers/"
	}

	return dir
}

func copyLayers(outputDir string) {
	layersDir := findLayersDir()

	layerBaseDir := filepath.Join(layersDir, "base")

	copyFile(layerBaseDir, outputDir, "style.css")
}

func FindTemplateDir() string {
	dir, _ := dirs.GetFirstExistingDirectory("template", []string{
		"../templates/",
		"/app/templates/",
		"/config/templates/",
	})

	return dir
}

func copyFile(fromDir string, toDir string, filename string) {
	contents, err := os.ReadFile(filepath.Join(fromDir, filename))

	if err != nil {
		log.Errorf("Error reading style.css: %v", err)
		return
	}

	os.WriteFile(filepath.Join(toDir, filename), contents, 0644)

	log.Infof("Copied %v to %v", filename, toDir)
}

func readDatafile(path string, buildconfigPath string) (any, error) {
	path = filepath.Join(buildconfigPath, path)

	content, err := os.ReadFile(path)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to read datafile: %v", path))
	}

	var data any

	err = yaml.Unmarshal(content, &data)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to unmarshal datafile: %v", path))
	}

	return data, nil
}
