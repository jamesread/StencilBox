package buildconfigs

import (
	"os"
	"path/filepath"
	"testing"
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
