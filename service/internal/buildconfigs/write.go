package buildconfigs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrInvalid wraps every validation failure, so callers can report it as bad input.
var ErrInvalid = errors.New("invalid")

// ErrExists is returned when a build config, file or data file name is already taken.
var ErrExists = errors.New("already exists")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

var (
	configNamePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]{0,63}$`)
	pathSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	// Data file names are template variables (eg {{ .links }}), so they must be identifiers.
	datafileNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)
	nonSlugChars        = regexp.MustCompile(`[^a-z0-9]+`)
)

// reservedOutputDirs are served by StencilBox itself, so a site there would be unreachable.
var reservedOutputDirs = map[string]bool{"webui": true, "api": true, "mcp": true}

// Slug turns a build config name into a file name stem, eg "My links" -> "my-links".
func Slug(name string) string {
	s := strings.Trim(nonSlugChars.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if s == "" {
		return "buildconfig"
	}
	return s
}

// ValidateName checks a build config name.
func ValidateName(name string) error {
	if !configNamePattern.MatchString(name) {
		return invalid("build config names are 1-64 letters, digits, spaces, dots, dashes or underscores, starting with a letter or digit")
	}
	return nil
}

// ValidateDatafileName checks a data file's name within a build config.
func ValidateDatafileName(name string) error {
	if !datafileNamePattern.MatchString(name) {
		return invalid("data file names start with a letter and use only letters, digits and underscores (templates refer to them by name)")
	}
	return nil
}

// ValidateOutputDir checks a new build config's output directory: a relative path of simple
// segments inside the output directory, that does not overlap another build config's output.
// Builds replace their whole output directory, so an overlap would delete another site.
func ValidateOutputDir(outputDir string, existing map[string]*BuildConfig) error {
	if outputDir == "" {
		return invalid("an output directory is required")
	}
	segments := strings.Split(outputDir, "/")
	for _, seg := range segments {
		if !pathSegmentPattern.MatchString(seg) {
			return invalid("output directory %q must be a relative path of letters, digits, dots, dashes and underscores", outputDir)
		}
	}
	if len(segments) > 4 {
		return invalid("output directory %q is nested too deeply", outputDir)
	}
	if reservedOutputDirs[strings.ToLower(segments[0])] {
		return invalid("output directory %q is used by StencilBox itself", outputDir)
	}
	if strings.HasSuffix(outputDir, "_tmp") {
		return invalid("output directories cannot end in _tmp; that suffix is used while building")
	}
	for name, bc := range existing {
		other := strings.Trim(filepath.ToSlash(filepath.Clean(bc.OutputDir)), "/")
		if other == "." {
			other = ""
		}
		if other == outputDir || other == "" ||
			strings.HasPrefix(outputDir, other+"/") || strings.HasPrefix(other, outputDir+"/") ||
			other == outputDir+"_tmp" {
			return invalid("output directory %q overlaps build config %q (output %q)", outputDir, name, bc.OutputDir)
		}
	}
	return nil
}

// NewBuildConfig describes a build config created through the API.
type NewBuildConfig struct {
	Name           string
	Template       string
	OutputDir      string
	BuildOnStartup bool
}

// Create writes a new build config file into dir and returns its path. It does not overwrite.
func Create(dir string, nb NewBuildConfig, existing map[string]*BuildConfig) (string, error) {
	nb.Name = strings.TrimSpace(nb.Name)
	nb.OutputDir = strings.Trim(strings.TrimSpace(nb.OutputDir), "/")
	if err := ValidateName(nb.Name); err != nil {
		return "", err
	}
	if _, taken := existing[nb.Name]; taken {
		return "", fmt.Errorf("%w: build config %q", ErrExists, nb.Name)
	}
	if err := ValidateOutputDir(nb.OutputDir, existing); err != nil {
		return "", err
	}

	path := filepath.Join(dir, Slug(nb.Name)+".yaml")
	// A struct, not a map, so the keys keep this order in the file.
	doc := struct {
		Name      string            `yaml:"name"`
		Template  string            `yaml:"template"`
		OutputDir string            `yaml:"outputdir"`
		OnStartup bool              `yaml:"onstartup"`
		Datafiles map[string]string `yaml:"datafiles"`
	}{nb.Name, nb.Template, nb.OutputDir, nb.BuildOnStartup, map[string]string{}}
	var buf bytes.Buffer
	buf.WriteString("# Created by StencilBox. You can edit this file by hand.\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return "", err
	}

	// O_EXCL: never replace a file that appeared since the names were checked.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("%w: file %s", ErrExists, filepath.Base(path))
	}
	if err != nil {
		return "", err
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", err
	}
	return path, f.Close()
}

// ResolveDatafilePath validates a data file path relative to the build configs directory and
// returns it cleaned (with forward slashes) and absolute.
func ResolveDatafilePath(dir, rel string) (string, string, error) {
	rel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
	if rel == "." || rel == "" || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", "", invalid("data file path %q must be relative to the build configs directory, and inside it", rel)
	}
	for _, seg := range strings.Split(rel, "/") {
		if !pathSegmentPattern.MatchString(seg) {
			return "", "", invalid("data file path %q must use letters, digits, dots, dashes and underscores", rel)
		}
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".yaml", ".yml", ".json":
	default:
		return "", "", invalid("data file path %q must end in .yaml, .yml or .json", rel)
	}
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if !InsideDir(dir, abs) {
		return "", "", invalid("data file path %q is outside the build configs directory", rel)
	}
	return rel, abs, nil
}

// InsideDir reports whether path is inside dir, after resolving symlinks in both (a path that
// does not exist yet is checked through its nearest existing parent).
func InsideDir(dir, path string) bool {
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return false
	}
	realPath, err := evalExisting(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(realDir, realPath)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// evalExisting resolves symlinks in the longest existing prefix of path.
func evalExisting(path string) (string, error) {
	path = filepath.Clean(path)
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path, nil
	}
	realParent, err := evalExisting(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(realParent, filepath.Base(path)), nil
}

// AddDatafile adds name: rel under datafiles in a build config file, keeping the rest of the
// file (including comments) as it is.
func AddDatafile(configPath, name, rel string) error {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", filepath.Base(configPath), err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return invalid("%s is not a YAML mapping", filepath.Base(configPath))
	}
	root := doc.Content[0]

	datafiles := mappingValue(root, "datafiles")
	if datafiles == nil {
		datafiles = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, scalar("datafiles"), datafiles)
	} else if datafiles.Kind != yaml.MappingNode {
		// eg "datafiles:" with no value, which parses as null.
		*datafiles = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}
	if mappingValue(datafiles, name) != nil {
		return fmt.Errorf("%w: data file %q", ErrExists, name)
	}
	datafiles.Style = 0 // block style, even if the file had "datafiles: {}"
	datafiles.Content = append(datafiles.Content, scalar(name), scalar(rel))

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	return WriteFileAtomic(configPath, buf.Bytes())
}

func scalar(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
}

func mappingValue(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// WriteFileAtomic replaces path via a temporary file in the same directory, so readers never
// see a half-written file. The existing file mode is kept.
func WriteFileAtomic(path string, content []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", path)
		}
		mode = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op after a successful rename

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
