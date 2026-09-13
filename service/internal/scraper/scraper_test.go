package scraper

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"", ""},
		{"  ", ""},
		{"xkcd.com", "https://xkcd.com"},
		{"https://xkcd.com", "https://xkcd.com"},
		{"http://example.com", "http://example.com"},
		{"  github.com/foo  ", "https://github.com/foo"},
	}

	for _, tt := range tests {
		if got := NormalizeURL(tt.raw); got != tt.want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

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

func TestIsFaviconMimeType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mime string
		want bool
	}{
		{"image/png", true},
		{"image/png; charset=utf-8", true},
		{"IMAGE/X-ICON", true},
		{"image/svg+xml", true},
		{"image/vnd.microsoft.icon", true},
		{"application/svg+xml", true},
		{"text/html", false},
		{"text/html; charset=utf-8", false},
		{"application/octet-stream", false},
		{"", false},
		{"application/json", false},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			t.Parallel()
			if got := isFaviconMimeType(tt.mime); got != tt.want {
				t.Fatalf("isFaviconMimeType(%q) = %v, want %v", tt.mime, got, tt.want)
			}
		})
	}
}

func TestResolveFaviconURLAbsolute(t *testing.T) {
	t.Parallel()

	got := resolveFaviconURL("/img/icon.png?v=2", "https://example.com/app/")
	want := "https://example.com/img/icon.png?v=2"
	if got != want {
		t.Fatalf("resolveFaviconURL() = %q, want %q", got, want)
	}
}

func TestDownloadFaviconRejectsNonImageMIME(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>not an icon</html>"))
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	_, err := DownloadFavicon(srv.URL+"/favicon.ico", dir, "site")
	if !errors.Is(err, ErrRejectedMime) {
		t.Fatalf("DownloadFavicon() err = %v, want ErrRejectedMime", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("rejected favicon should not be saved, found %d files", len(entries))
	}
}

func TestFindAndDownloadFaviconFallsThroughToNextCandidate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><link rel="icon" href="/bad.ico">`))
	})
	mux.HandleFunc("/bad.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html>nope</html>"))
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG\r\n\x1a\n"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	filename, err := FindAndDownloadFavicon(srv.URL, dir, "site")
	if err != nil {
		t.Fatalf("FindAndDownloadFavicon() err = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "\x89PNG") {
		t.Fatalf("saved file is not the PNG fallback")
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
