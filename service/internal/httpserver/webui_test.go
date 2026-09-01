package httpserver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWebuiFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	index := filepath.Join(dir, "index.html")
	assetDir := filepath.Join(dir, "assets")
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	asset := filepath.Join(assetDir, "app.js")
	if err := os.WriteFile(index, []byte("spa"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(asset, []byte("js"), 0644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	tests := []struct {
		name        string
		requestPath string
		want        string
	}{
		{name: "root serves index", requestPath: "", want: index},
		{name: "slash serves index", requestPath: "/", want: index},
		{name: "spa route serves index", requestPath: "/build-config/homepage", want: index},
		{name: "existing asset is served", requestPath: "/assets/app.js", want: asset},
		{name: "path traversal falls back to index", requestPath: "../secret", want: index},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolveWebuiFile(dir, tt.requestPath)
			if got != tt.want {
				t.Fatalf("resolveWebuiFile(%q) = %q, want %q", tt.requestPath, got, tt.want)
			}
		})
	}
}
