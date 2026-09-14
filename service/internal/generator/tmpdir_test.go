package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemporaryOutputDir(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	got, err := TemporaryOutputDir(base, "homepage")
	if err != nil {
		t.Fatalf("TemporaryOutputDir() err = %v", err)
	}
	want := filepath.Join(base, "homepage_tmp")
	if got != want {
		t.Fatalf("TemporaryOutputDir() = %q, want %q", got, want)
	}
}

func TestTemporaryOutputDirRejectsEscape(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	_, err := TemporaryOutputDir(base, filepath.Join("..", "outside"))
	if err == nil {
		t.Fatal("expected error for path outside base")
	}
}

func TestTemporaryOutputDirRejectsEmptyOutputDir(t *testing.T) {
	t.Parallel()

	_, err := TemporaryOutputDir(t.TempDir(), "  ")
	if err == nil {
		t.Fatal("expected error for empty output dir")
	}
}

func TestClearTemporaryOutputDirRemovesContents(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	tmp, err := TemporaryOutputDir(base, "homepage")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "icons"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "icons", "site.png"), []byte("png"), 0644); err != nil {
		t.Fatal(err)
	}

	cleared, err := ClearTemporaryOutputDir(base, "homepage")
	if err != nil {
		t.Fatalf("ClearTemporaryOutputDir() err = %v", err)
	}
	if cleared != tmp {
		t.Fatalf("cleared path = %q, want %q", cleared, tmp)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("tmp dir still exists: %v", err)
	}
}

func TestClearTemporaryOutputDirMissingIsOK(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	cleared, err := ClearTemporaryOutputDir(base, "homepage")
	if err != nil {
		t.Fatalf("ClearTemporaryOutputDir() err = %v", err)
	}
	if !strings.HasSuffix(cleared, "homepage_tmp") {
		t.Fatalf("cleared path = %q, want suffix homepage_tmp", cleared)
	}
}
