package generator

import (
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/golure/pkg/dirs"
	"github.com/jamesread/golure/pkg/git"
	"errors"

	"fmt"
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

func Generate(baseOutputDir string, cfg *buildconfigs.BuildConfig) *BuildStatus {
	buildStatus := &BuildStatus{
		IsError: false,
	}

	finalOutputDir := filepath.Join(baseOutputDir, cfg.OutputDir)

	log.WithFields(log.Fields{
		"name":	   cfg.Name,
		"outputDir": cfg.OutputDir,
		"finalOutputDir": finalOutputDir,
	}).Infof("Starting build for project")

	os.MkdirAll(finalOutputDir, 0755)

	indexPath := filepath.Join(FindTemplateDir(), cfg.Template, "index.html")

	tmpl, err := template.ParseFiles(indexPath)

	if err != nil {
		buildStatus.IsError = true
		buildStatus.Message = "Failed to parse template: " + err.Error()

		return buildStatus
	}

	datafiles := make(map[string]any)

	for name, path := range cfg.Datafiles {
		log.Infof("Adding datafile %v with path %v", name, path)

		dat, err := readDatafile(path, filepath.Dir(cfg.Path))

		if err != nil {
			buildStatus.IsError = true
			buildStatus.Message = "Error reading datafile " + name + ": " + err.Error()
			return buildStatus
		}

		if dat == nil {
			continue
		}

		datafiles[name] = dat
	}

	log.Infof("Datafiles loaded: %+v", datafiles)

	outfile, err := os.Create(finalOutputDir + "/index.html")

	if err != nil {
		log.Errorf("Error creating output file: %v", err)
		return buildStatus
	}

	err = tmpl.Execute(outfile, datafiles)

	if err != nil {
		buildStatus.IsError = true
		buildStatus.Message = "Error executing template: " + err.Error()
		return buildStatus
	}

	copyLayers(finalOutputDir)
	cloneRepos(cfg.Repos, finalOutputDir)

	buildStatus.Message = "Build completed successfully"
	buildStatus.IsError = false
	buildStatus.BuildUrlBase = getBuildUrlBase()
	buildStatus.OutputSizeHumanReadable = getDirectorySizeHumanReadable(finalOutputDir)

	return buildStatus
}

func getDirectorySizeHumanReadable(dir string) string {
	// Get the size of the getDirectorySizeHumanReadable
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

func cloneRepos(repos []buildconfigs.GitRepo, outputDir string) {
	for _, repo := range repos {
		res := git.CloneOrPull(repo.URL, outputDir)

		if res.WasCloned {
			log.Infof("Cloned repo %s to %s", repo.URL, outputDir)
		} else {
			log.Infof("Pulled repo %s in %s", repo.URL, outputDir)
		}
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

func copyLayers(finalOutputDir string) {
	layersDir := findLayersDir()

	layerBaseDir := filepath.Join(layersDir, "base")

	copyFile(layerBaseDir, finalOutputDir, "style.css")
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
