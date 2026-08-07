package config

import (
	"path/filepath"
	"testing"
)

func TestSetConfigDirOverridesPath(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(func() { SetConfigDir("") })

	got := GetConfigPath()
	want := filepath.Join(dir, "config.yaml")
	if got != want {
		t.Errorf("GetConfigPath() = %q, want %q", got, want)
	}

	if OverrideDir() != dir {
		t.Errorf("OverrideDir() = %q, want %q", OverrideDir(), dir)
	}
}
