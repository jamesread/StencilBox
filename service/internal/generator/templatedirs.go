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

	ret = appendTemplateDirs(ret, seen, customTemplateDirCandidates(), TemplateSourceCustom)
	ret = appendTemplateDirs(ret, seen, builtinTemplateDirCandidates(), TemplateSourceBuiltin)

	return ret
}

// appendTemplateDirs adds the candidates that are existing directories not seen before.
func appendTemplateDirs(dirs []TemplateDir, seen map[string]bool, candidates []string, source string) []TemplateDir {
	for _, candidate := range candidates {
		abs, ok := resolveTemplateDir(candidate)
		if !ok || seen[abs] {
			continue
		}

		seen[abs] = true
		dirs = append(dirs, TemplateDir{Path: abs, Source: source})
	}

	return dirs
}

// resolveTemplateDir returns the absolute path of candidate, and whether it is a directory.
func resolveTemplateDir(candidate string) (string, bool) {
	if strings.TrimSpace(candidate) == "" {
		return "", false
	}

	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}

	info, err := os.Stat(abs)
	return abs, err == nil && info.IsDir()
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
		found, ok, err := lookupTemplate(dir, name)
		if err != nil {
			return TemplateDir{}, err
		}

		if ok {
			return found, nil
		}
	}

	return TemplateDir{}, fmt.Errorf("template %q not found", name)
}

// lookupTemplate finds the named template in a template directory, and reports whether it is there.
func lookupTemplate(dir TemplateDir, name string) (TemplateDir, bool, error) {
	templateDir := filepath.Join(dir.Path, name)

	if !isUnderDir(dir.Path, templateDir) || templateDir == dir.Path {
		return TemplateDir{}, false, fmt.Errorf("invalid template name %q", name)
	}

	_, err := os.Stat(filepath.Join(templateDir, "index.html"))
	return TemplateDir{Path: templateDir, Source: dir.Source}, err == nil, nil
}
