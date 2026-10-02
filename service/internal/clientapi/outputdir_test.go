package clientapi

import (
	"os"
	"testing"
)

func TestFindOutputDirEnvOverride(t *testing.T) {
	t.Setenv("STENCILBOX_OUTPUT_DIR", "/custom/output")
	got := findOutputDir()
	if got != "/custom/output" {
		t.Fatalf("findOutputDir() = %q, want /custom/output", got)
	}
}

func TestDefaultOutputDirWithoutConfigRoot(t *testing.T) {
	if _, err := os.Stat("/config"); err == nil {
		t.Skip("/config exists on this machine")
	}
	if got := defaultOutputDir(); got != "./sb-output" {
		t.Fatalf("defaultOutputDir() = %q, want ./sb-output", got)
	}
}

func TestFindOutputDirUsesExistingPathFromEnv(t *testing.T) {
	outputDir := t.TempDir()
	t.Setenv("STENCILBOX_OUTPUT_DIR", outputDir)
	got := findOutputDir()
	if got != outputDir {
		t.Fatalf("findOutputDir() = %q, want %q", got, outputDir)
	}
}
