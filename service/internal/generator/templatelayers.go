package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jamesread/golure/pkg/dirs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// templateMetadata is the part of a template's metadata.yaml the generator reads.
type templateMetadata struct {
	// Layers are extra files copied next to the built index.html, eg "femtocrank".
	Layers []string `yaml:"layers"`
}

var layerNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// npmLayers are layers that come from an npm package installed with the build tools
// (var/tools, /app/tools in the container), so Renovate keeps them up to date. Each maps the
// package's files to the names they get in the output.
var npmLayers = map[string]struct {
	pkg   string
	files map[string]string
}{
	"femtocrank": {
		pkg:   "femtocrank",
		files: map[string]string{"style.css": "femtocrank.css", "dark.css": "femtocrank-dark.css"},
	},
}

// readTemplateLayers returns the layers a template's metadata.yaml asks for.
func readTemplateLayers(templateDir string) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(templateDir, "metadata.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var md templateMetadata
	if err := yaml.Unmarshal(raw, &md); err != nil {
		return nil, fmt.Errorf("parse metadata.yaml: %w", err)
	}
	for _, name := range md.Layers {
		if !layerNamePattern.MatchString(name) {
			return nil, fmt.Errorf("layer name %q in metadata.yaml is not valid", name)
		}
	}
	return md.Layers, nil
}

// copyTemplateLayers copies the layers a template asks for into the output directory.
func copyTemplateLayers(templateDir, outputDir string) error {
	layers, err := readTemplateLayers(templateDir)
	if err != nil {
		return err
	}
	for _, name := range layers {
		if err := copyLayer(name, outputDir); err != nil {
			return fmt.Errorf("layer %s: %w", name, err)
		}
		log.Infof("Copied layer %s to %s", name, outputDir)
	}
	return nil
}

func copyLayer(name, outputDir string) error {
	if npm, ok := npmLayers[name]; ok {
		pkgDir := findToolsPackage(npm.pkg)
		if pkgDir == "" {
			return fmt.Errorf("npm package %s is not installed with the build tools (run make -C var/tools)", npm.pkg)
		}
		for from, to := range npm.files {
			if err := copyRegularFile(filepath.Join(pkgDir, from), filepath.Join(outputDir, to)); err != nil {
				return err
			}
		}
		return nil
	}

	// Otherwise a directory under layers/, copied flat (no subdirectories).
	dir := filepath.Join(findLayersDir(), name)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("no such layer: %w", err)
	}
	for _, e := range entries {
		if e.Type().IsRegular() {
			if err := copyRegularFile(filepath.Join(dir, e.Name()), filepath.Join(outputDir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// findToolsPackage returns the directory of an npm package installed with the build tools.
func findToolsPackage(pkg string) string {
	candidates := []string{
		filepath.Join("/app/tools/node_modules", pkg),
		filepath.Join("../var/tools/node_modules", pkg),
	}
	if toolsDir := os.Getenv("STENCILBOX_TOOLS_DIR"); toolsDir != "" {
		candidates = append([]string{filepath.Join(toolsDir, "node_modules", pkg)}, candidates...)
	}
	dir, err := dirs.GetFirstExistingDirectory("tools package "+pkg, candidates)
	if err != nil {
		return ""
	}
	return dir
}

func copyRegularFile(from, to string) error {
	contents, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, contents, 0o644)
}
