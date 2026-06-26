package scraper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImageFileExt(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"ih.apps.moo.teratan.net", ""},
		{"ih.apps.moo.teratan.net.svg", ".svg"},
		{"https://example.com/favicon.svg", ".svg"},
		{"https://example.com/favicon.svg?v=1", ".svg"},
		{"github.com", ""},
		{"icon.png", ".png"},
		{"icon.ico", ".ico"},
	}

	for _, tt := range tests {
		if got := imageFileExt(tt.path); got != tt.want {
			t.Errorf("imageFileExt(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestDownloadFaviconSVGExtension(t *testing.T) {
	dir := t.TempDir()
	filename, err := DownloadFavicon("https://ih.apps.moo.teratan.net/favicon.svg", dir, "ih.apps.moo.teratan.net")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filename, ".svg") {
		t.Fatalf("expected .svg extension, got %q", filename)
	}
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "<svg") {
		t.Fatalf("expected svg content")
	}
}
