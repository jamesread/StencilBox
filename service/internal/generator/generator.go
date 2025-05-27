package generator

import (
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/golure/pkg/dirs"

	"os"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"

	"path/filepath"

	"text/template"
)

func Generate(baseOutputDir string, cfg *buildconfigs.BuildConfig) {
	finalOutputDir := filepath.Join(baseOutputDir, cfg.OutputDir)

	log.WithFields(log.Fields{
		"name":	   cfg.Name,
		"outputDir": cfg.OutputDir,
		"finalOutputDir": finalOutputDir,
	}).Infof("Starting build for project")

	os.MkdirAll(finalOutputDir, 0755)

	tmpl, err := template.ParseFiles(findTemplateDir() + cfg.Template + "/index.html")

	if err != nil {
		log.Errorf("Error parsing template %v: %v", cfg.Template, err)
		return
	}

	datafiles := make(map[string]any)

	for name, path := range cfg.Datafiles {
		log.Infof("Adding datafile %v with path %v", name, path)

		dat := readDatafile(path)

		if dat == nil {
			continue
		}

		datafiles[name] = dat
	}

	log.Infof("Datafiles loaded: %+v", datafiles)

	outfile, err := os.Create(finalOutputDir + "/index.html")

	if err != nil {
		log.Errorf("Error creating output file: %v", err)
		return
	}

	err = tmpl.Execute(outfile, datafiles)

	if err != nil {
		log.Errorf("Error executing template: %v", err)
		return
	}

	copyAssets(finalOutputDir)
}

func findTemplateDir() string {
	dir, _ := dirs.GetFirstExistingDirectory([]string{
		"../templates/",
		"/config/templates/",
	})

	return dir
}

func copyAssets(outputDir string) {
	contents, err := os.ReadFile("../style.css")

	if err != nil {
		log.Errorf("Error reading style.css: %v", err)
		return
	}

	os.WriteFile(outputDir + "/style.css", contents, 0644)
}

func readDatafile(path string) any {
	content, err := os.ReadFile(path)

	if err != nil {
		log.Errorf("Error reading datafile %v: %v", path, err)
		return nil
	}

	var data any

	err = yaml.Unmarshal(content, &data)

	if err != nil {
		log.Errorf("Error unmarshalling datafile %v: %v", path, err)
		return nil
	}

	return data
}
