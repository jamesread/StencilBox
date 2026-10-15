package buildconfigs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesread/StencilBox/internal/config"
)

func TestBuildOnStartupDefault(t *testing.T) {
	cfg := &BuildConfig{}

	if !cfg.BuildOnStartup() {
		t.Fatal("expected BuildOnStartup to default to true")
	}
}

func TestBuildOnStartupExplicitValues(t *testing.T) {
	falseVal := false
	trueVal := true

	if (&BuildConfig{OnStartup: &falseVal}).BuildOnStartup() {
		t.Fatal("expected onstartup: false to disable startup builds")
	}

	if !(&BuildConfig{OnStartup: &trueVal}).BuildOnStartup() {
		t.Fatal("expected onstartup: true to enable startup builds")
	}
}

func TestReadBuildConfigOnStartup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "buildconfig-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "defaults to true when omitted",
			content: `name: example
template: links-homepage
outputdir: example
`,
			expected: true,
		},
		{
			name: "explicit true",
			content: `name: example
template: links-homepage
outputdir: example
onstartup: true
`,
			expected: true,
		},
		{
			name: "explicit false",
			content: `name: example
template: links-homepage
outputdir: example
onstartup: false
`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(tmpDir, tt.name+".yaml")
			if err := os.WriteFile(file, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg := readBuildConfig(file)
			if cfg == nil {
				t.Fatal("expected build config to be parsed")
			}

			if cfg.BuildOnStartup() != tt.expected {
				t.Fatalf("BuildOnStartup() = %v, want %v", cfg.BuildOnStartup(), tt.expected)
			}
		})
	}
}

func TestReadConfigFilesNameFallbackAndDuplicates(t *testing.T) {
	dir := t.TempDir()
	config.SetConfigDir(dir)
	defer config.SetConfigDir("")

	bcDir := filepath.Join(dir, "buildconfigs")
	if err := os.MkdirAll(bcDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"noname.yaml": "template: debug\noutputdir: a\n",
		"one.yaml":    "name: Same\ntemplate: debug\noutputdir: b\n",
		"two.yaml":    "name: Same\ntemplate: debug\noutputdir: c\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(bcDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got := ReadConfigFiles()

	if len(got) != 3 {
		t.Fatalf("expected 3 configs, got %d: %v", len(got), got)
	}
	if bc, ok := got["noname"]; !ok || bc.OutputDir != "a" {
		t.Errorf("expected unnamed config to fall back to filename, got %v", got)
	}
	if bc, ok := got["Same"]; !ok || bc.OutputDir != "b" {
		t.Errorf("expected first config to keep its name, got %v", got)
	}
	if bc, ok := got["two.yaml"]; !ok || bc.OutputDir != "c" {
		t.Errorf("expected duplicate to be renamed to its filename, got %v", got)
	}
}
