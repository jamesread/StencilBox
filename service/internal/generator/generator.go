package generator

import (
	"github.com/jamesread/StencilBox/internal/config"

	log "github.com/sirupsen/logrus"
	"os"

	"gopkg.in/yaml.v3"

	"text/template"
)

func Generate(cfg *config.Config) {
	log.Infof("Starting build for project %v", cfg.OutputDir)

	tmpl, err := template.ParseFiles("../templates/" + cfg.Template + "/index.html")

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

	cfg.OutputDir = "sb-output/"

	outfile, err := os.Create(cfg.OutputDir + "/index.html")

	if err != nil {
		log.Errorf("Error creating output file: %v", err)
		return
	}

	err = tmpl.Execute(outfile, datafiles)

	if err != nil {
		log.Errorf("Error executing template: %v", err)
		return
	}

	copyAssets(cfg.OutputDir)
}

func copyAssets(outputDir string) {
	outputDir = "sb-output/"

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
