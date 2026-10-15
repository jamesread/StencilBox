package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jamesread/StencilBox/internal/config"
)

const (
	TemplateSourceBuiltin = "built-in"
	TemplateSourceCustom  = "custom"
)

// TemplateDir is a directory containing one subdirectory per template.
type TemplateDir struct {
	Path   string
	Source string
}

func customTemplateDirCandidates() []string {
	candidates := []string{os.Getenv("STENCILBOX_TEMPLATES_DIR")}

	if dir := config.OverrideDir(); dir != "" {
		candidates = append(candidates, filepath.Join(dir, "templates"))
	}

	return append(candidates, "/config/templates/")
}

func builtinTemplateDirCandidates() []string {
	return []string{
		"../templates/",
		"/app/templates/",
	}
}

// FindTemplateDirs returns every existing template directory, custom directories first so
// that a custom template overrides a built-in template of the same name.
func FindTemplateDirs() []TemplateDir {
	var ret []TemplateDir
	seen := make(map[string]bool)

	add := func(candidates []string, source string) {
		for _, candidate := range candidates {
			if strings.TrimSpace(candidate) == "" {
				continue
			}

			abs, err := filepath.Abs(candidate)
			if err != nil || seen[abs] {
				continue
			}

			if info, err := os.Stat(abs); err != nil || !info.IsDir() {
				continue
			}

			seen[abs] = true
			ret = append(ret, TemplateDir{Path: abs, Source: source})
		}
	}

	add(customTemplateDirCandidates(), TemplateSourceCustom)
	add(builtinTemplateDirCandidates(), TemplateSourceBuiltin)

	return ret
}

// CustomTemplatesDir returns the directory where users should add their own templates: the
// first existing custom directory, or the first candidate if none exist yet.
func CustomTemplatesDir() string {
	for _, dir := range FindTemplateDirs() {
		if dir.Source == TemplateSourceCustom {
			return dir.Path
		}
	}

	for _, candidate := range customTemplateDirCandidates() {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}

	return ""
}

// FindTemplate returns the directory of the named template, searching custom template
// directories before built-in ones.
func FindTemplate(name string) (TemplateDir, error) {
	if strings.TrimSpace(name) == "" {
		return TemplateDir{}, fmt.Errorf("no template specified")
	}

	for _, dir := range FindTemplateDirs() {
		templateDir := filepath.Join(dir.Path, name)

		if !isUnderDir(dir.Path, templateDir) || templateDir == dir.Path {
			return TemplateDir{}, fmt.Errorf("invalid template name %q", name)
		}

		if _, err := os.Stat(filepath.Join(templateDir, "index.html")); err == nil {
			return TemplateDir{Path: templateDir, Source: dir.Source}, nil
		}
	}

	return TemplateDir{}, fmt.Errorf("template %q not found", name)
}
