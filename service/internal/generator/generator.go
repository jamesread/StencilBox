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
		buildStatus.IsError = true
		buildStatus.Message = "Failed to parse template: " + err.Error()
        return
	}

	datafiles := make(map[string]any)

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

		datafiles[name] = dat
	}

	log.Infof("Datafiles loaded: %+v", datafiles)

	outfile, err := os.Create(temporaryOutputDir + "/index.html")

	if err != nil {
		log.Errorf("Error creating output file: %v", err)
		return
	}

	err = tmpl.Execute(outfile, datafiles)

	if err != nil {
		buildStatus.IsError = true
		buildStatus.Message = "Error executing template: " + err.Error()
		return
	}

	repoStorageDir := getRepoStorageDir(cfg)

	copyLayers(temporaryOutputDir)
	cloneRepos(cfg.Repos, repoStorageDir, updateChannel)

	lnRepos(temporaryOutputDir, repoStorageDir)

	updateChannel <- "Running Vite build..."

	runVite(temporaryOutputDir, finalOutputDir, cfg.OutputDir)

	updateChannel <- "Build completed!"

	buildStatus.Message = "Build completed successfully"
	buildStatus.IsError = false
	buildStatus.BuildUrlBase = getBuildUrlBase()
	buildStatus.OutputSizeHumanReadable = getDirectorySizeHumanReadable(finalOutputDir)

    close(updateChannel)

	return
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
	log.Info("Running Vite build...")
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
