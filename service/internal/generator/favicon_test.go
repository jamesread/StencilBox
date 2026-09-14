package generator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesread/StencilBox/internal/scraper"
)

func TestLinkFaviconBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		linkMap  map[string]any
		expected string
	}{
		{
			name: "uses url when url_internal is absent",
			linkMap: map[string]any{
				"url": "https://example.com",
			},
			expected: "https://example.com",
		},
		{
			name: "uses url_internal when set",
			linkMap: map[string]any{
				"url":          "https://grafana.example.com",
				"url_internal": "http://grafana.internal:3000",
			},
			expected: "http://grafana.internal:3000",
		},
		{
			name: "falls back to url when url_internal is empty",
			linkMap: map[string]any{
				"url":          "https://example.com",
				"url_internal": "",
			},
			expected: "https://example.com",
		},
		{
			name:     "returns empty when no url fields are set",
			linkMap:  map[string]any{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := linkFaviconBaseURL(tt.linkMap); got != tt.expected {
				t.Fatalf("linkFaviconBaseURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNormalizeLinksDataURLs(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"categories": []any{
			map[string]any{
				"title": "tech",
				"links": []any{
					map[string]any{"url": "xkcd.com", "title": "XKCD"},
					map[string]any{"url": "https://github.com", "title": "GitHub"},
				},
			},
		},
		"links": []any{
			map[string]any{"url": "google.com", "title": "Google"},
		},
	}

	normalizeLinksDataURLs(data)

	categories := data["categories"].([]any)
	catLinks := categories[0].(map[string]any)["links"].([]any)
	if got := catLinks[0].(map[string]any)["url"]; got != "https://xkcd.com" {
		t.Fatalf("category link url = %q, want https://xkcd.com", got)
	}
	if got := catLinks[1].(map[string]any)["url"]; got != "https://github.com" {
		t.Fatalf("category link url = %q, want https://github.com", got)
	}

	flatLinks := data["links"].([]any)
	if got := flatLinks[0].(map[string]any)["url"]; got != "https://google.com" {
		t.Fatalf("flat link url = %q, want https://google.com", got)
	}
}

func drainLogChan(ch chan string) []string {
	n := len(ch)
	out := make([]string, 0, n)
	for range n {
		out = append(out, <-ch)
	}
	return out
}

func testLinksData(pageURL string) map[string]any {
	return map[string]any{
		"categories": []any{
			map[string]any{
				"links": []any{
					map[string]any{"url": pageURL, "title": "Local"},
				},
			},
		},
	}
}

func TestProcessLinksWithFaviconsLogsIconURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><link rel="icon" href="/brand/logo.svg">`))
	})
	mux.HandleFunc("/brand/logo.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	ch := make(chan string, 16)
	_, err := processLinksWithFavicons(context.Background(), testLinksData(srv.URL), t.TempDir(), ch)
	if err != nil {
		t.Fatalf("processLinksWithFavicons() err = %v", err)
	}

	logs := strings.Join(drainLogChan(ch), "\n")
	want := "Found favicon: " + srv.URL + "/brand/logo.svg (image/svg+xml)"
	if !strings.Contains(logs, want) {
		t.Fatalf("build log missing icon URL and mime type %q\n%s", want, logs)
	}
}

func TestFaviconFoundMessageIncludesMimeType(t *testing.T) {
	t.Parallel()

	got := faviconFoundMessage(scraper.DownloadedFavicon{
		SourceURL: "https://example.com/icon.png",
		MimeType:  "image/png",
	})
	want := "Found favicon: https://example.com/icon.png (image/png)"
	if got != want {
		t.Fatalf("faviconFoundMessage() = %q, want %q", got, want)
	}
}

func TestProcessLinksWithFaviconsLogsFetchFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	ch := make(chan string, 16)
	_, err := processLinksWithFavicons(context.Background(), testLinksData(srv.URL), t.TempDir(), ch)
	if err != nil {
		t.Fatalf("processLinksWithFavicons() err = %v", err)
	}

	logs := strings.Join(drainLogChan(ch), "\n")
	if !strings.Contains(logs, "No favicon found for "+srv.URL) {
		t.Fatalf("build log missing fetch failure\n%s", logs)
	}
	if !strings.Contains(logs, srv.URL+"/favicon.ico") {
		t.Fatalf("build log missing tried icon URL\n%s", logs)
	}
}

func TestProcessLinksWithFaviconsLogsCachedIcon(t *testing.T) {
	pageURL := "http://127.0.0.1:8090/"
	outDir := t.TempDir()
	iconsDir := filepath.Join(outDir, "icons")
	if err := os.MkdirAll(iconsDir, 0755); err != nil {
		t.Fatal(err)
	}
	cachedName := sanitizeFilename(pageURL) + ".png"
	if err := os.WriteFile(filepath.Join(iconsDir, cachedName), []byte("png"), 0644); err != nil {
		t.Fatal(err)
	}

	ch := make(chan string, 16)
	_, err := processLinksWithFavicons(context.Background(), testLinksData(pageURL), outDir, ch)
	if err != nil {
		t.Fatalf("processLinksWithFavicons() err = %v", err)
	}

	logs := strings.Join(drainLogChan(ch), "\n")
	want := "Using cached favicon for " + pageURL + ": icons/" + cachedName
	if !strings.Contains(logs, want) {
		t.Fatalf("build log missing cached icon path %q\n%s", want, logs)
	}
}
