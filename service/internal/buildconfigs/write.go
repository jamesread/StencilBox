package buildconfigs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
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
	if err := validateOutputSegments(outputDir); err != nil {
		return err
	}
	if strings.HasSuffix(outputDir, "_tmp") {
		return invalid("output directories cannot end in _tmp; that suffix is used while building")
	}
	return checkOutputOverlap(outputDir, existing)
}

func validateOutputSegments(outputDir string) error {
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
	return nil
}

func checkOutputOverlap(outputDir string, existing map[string]*BuildConfig) error {
	for name, bc := range existing {
		if outputsOverlap(outputDir, normalizeOutputDir(bc.OutputDir)) {
			return invalid("output directory %q overlaps build config %q (output %q)", outputDir, name, bc.OutputDir)
		}
	}
	return nil
}

func normalizeOutputDir(dir string) string {
	other := strings.Trim(filepath.ToSlash(filepath.Clean(dir)), "/")
	if other == "." {
		return ""
	}
	return other
}

func outputsOverlap(outputDir, other string) bool {
	if other == "" || other == outputDir || other == outputDir+"_tmp" {
		return true
	}
	return strings.HasPrefix(outputDir, other+"/") || strings.HasPrefix(other, outputDir+"/")
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
	if err := validateNewBuildConfig(nb, existing); err != nil {
		return "", err
	}

	path := filepath.Join(dir, Slug(nb.Name)+".yaml")
	content, err := encodeNewBuildConfig(nb)
	if err != nil {
		return "", err
	}
	return path, writeNewFile(path, content)
}

func validateNewBuildConfig(nb NewBuildConfig, existing map[string]*BuildConfig) error {
	if err := ValidateName(nb.Name); err != nil {
		return err
	}
	if _, taken := existing[nb.Name]; taken {
		return fmt.Errorf("%w: build config %q", ErrExists, nb.Name)
	}
	return ValidateOutputDir(nb.OutputDir, existing)
}

// encodeNewBuildConfig builds the document as a yaml.Node, so the keys keep this order in the file.
func encodeNewBuildConfig(nb NewBuildConfig) ([]byte, error) {
	doc := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		scalar("name"), scalar(nb.Name),
		scalar("template"), scalar(nb.Template),
		scalar("outputdir"), scalar(nb.OutputDir),
		scalar("onstartup"), {Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(nb.BuildOnStartup)},
		scalar("datafiles"), {Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle},
	}}

	var buf bytes.Buffer
	buf.WriteString("# Created by StencilBox. You can edit this file by hand.\n")
	if err := encodeYAML(&buf, doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeYAML(buf *bytes.Buffer, v any) error {
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return enc.Close()
}

// writeNewFile creates path with content. O_EXCL: never replace a file that appeared since
// the names were checked.
func writeNewFile(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("%w: file %s", ErrExists, filepath.Base(path))
	}
	if err != nil {
		return err
	}
	return writeAndClose(f, path, content)
}

func writeAndClose(f *os.File, path string, content []byte) error {
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}

// ResolveDatafilePath validates a data file path relative to the build configs directory and
// returns it cleaned (with forward slashes) and absolute.
func ResolveDatafilePath(dir, rel string) (string, string, error) {
	rel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
	if err := validateRelDatafilePath(rel); err != nil {
		return "", "", err
	}
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if !InsideDir(dir, abs) {
		return "", "", invalid("data file path %q is outside the build configs directory", rel)
	}
	return rel, abs, nil
}

func isEscapingPath(rel string) bool {
	return rel == "." || rel == "" || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, "../")
}

func validateRelDatafilePath(rel string) error {
	if isEscapingPath(rel) {
		return invalid("data file path %q must be relative to the build configs directory, and inside it", rel)
	}
	for seg := range strings.SplitSeq(rel, "/") {
		if !pathSegmentPattern.MatchString(seg) {
			return invalid("data file path %q must use letters, digits, dots, dashes and underscores", rel)
		}
	}
	if !slices.Contains([]string{".yaml", ".yml", ".json"}, strings.ToLower(filepath.Ext(rel))) {
		return invalid("data file path %q must end in .yaml, .yml or .json", rel)
	}
	return nil
}

// InsideDir reports whether path is inside dir, after resolving symlinks in both (a path that
// does not exist yet is checked through its nearest existing parent).
func InsideDir(dir, path string) bool {
	rel, err := relativeToRealDir(dir, path)
	if err != nil {
		return false
	}
	return relIsInside(rel)
}

func relativeToRealDir(dir, path string) (string, error) {
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	realPath, err := evalExisting(path)
	if err != nil {
		return "", err
	}
	return filepath.Rel(realDir, realPath)
}

func relIsInside(rel string) bool {
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

	doc, root, err := parseConfigDocument(raw, filepath.Base(configPath))
	if err != nil {
		return err
	}

	datafiles := ensureDatafilesNode(root)
	if mappingValue(datafiles, name) != nil {
		return fmt.Errorf("%w: data file %q", ErrExists, name)
	}
	datafiles.Style = 0 // block style, even if the file had "datafiles: {}"
	datafiles.Content = append(datafiles.Content, scalar(name), scalar(rel))

	var buf bytes.Buffer
	if err = encodeYAML(&buf, doc); err != nil {
		return err
	}
	return WriteFileAtomic(configPath, buf.Bytes())
}

// parseConfigDocument parses a build config file, returning the document and its root mapping.
func parseConfigDocument(raw []byte, filename string) (*yaml.Node, *yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", filename, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil, invalid("%s is not a YAML mapping", filename)
	}
	return &doc, doc.Content[0], nil
}

// ensureDatafilesNode returns the datafiles mapping of root, creating it when missing.
func ensureDatafilesNode(root *yaml.Node) *yaml.Node {
	datafiles := mappingValue(root, "datafiles")
	if datafiles == nil {
		datafiles = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, scalar("datafiles"), datafiles)
		return datafiles
	}
	if datafiles.Kind != yaml.MappingNode {
		// eg "datafiles:" with no value, which parses as null.
		*datafiles = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}
	return datafiles
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

// fileModeOf returns the permissions of path and whether it is a regular file (or does not
// exist yet, in which case the default mode applies).
func fileModeOf(path string) (os.FileMode, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return 0o644, true
	}
	return info.Mode().Perm(), info.Mode().IsRegular()
}

// WriteFileAtomic replaces path via a temporary file in the same directory, so readers never
// see a half-written file. The existing file mode is kept.
func WriteFileAtomic(path string, content []byte) error {
	mode, regular := fileModeOf(path)
	if !regular {
		return fmt.Errorf("%s is not a regular file", path)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op after a successful rename

	if err = writeSyncClose(tmp, content); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func writeSyncClose(f *os.File, content []byte) error {
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
