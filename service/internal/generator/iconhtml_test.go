package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderLinkIconHTML(t *testing.T) {
	dir := t.TempDir()
	iconDir := filepath.Join(dir, "icons")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		t.Fatal(err)
	}

	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><circle cx="5" cy="5" r="4"/></svg>`
	if err := os.WriteFile(filepath.Join(iconDir, "test.svg"), []byte(svg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(iconDir, "legacy.net"), []byte(svg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(iconDir, "photo.png"), []byte{0x89, 0x50, 0x4e, 0x47}, 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("svg by extension is inlined", func(t *testing.T) {
		html := renderLinkIconHTML(dir, "icons/test.svg", "favicon", "")
		if strings.Contains(html, "<img") {
			t.Fatalf("expected inline svg, got img tag: %s", html)
		}
		if !strings.Contains(html, "<svg") {
			t.Fatalf("expected svg content: %s", html)
		}
		if !strings.Contains(html, `class="favicon link-icon-svg"`) {
			t.Fatalf("expected favicon wrapper class: %s", html)
		}
	})

	t.Run("svg content without extension is inlined", func(t *testing.T) {
		html := renderLinkIconHTML(dir, "icons/legacy.net", "", "")
		if strings.Contains(html, "<img") {
			t.Fatalf("expected inline svg, got img tag: %s", html)
		}
		if !strings.Contains(html, "<svg") {
			t.Fatalf("expected svg content: %s", html)
		}
	})

	t.Run("png uses img tag", func(t *testing.T) {
		html := renderLinkIconHTML(dir, "icons/photo.png", "favicon", "")
		if !strings.Contains(html, `<img src="icons/photo.png"`) {
			t.Fatalf("expected img tag: %s", html)
		}
	})

	t.Run("remote svg uses object tag", func(t *testing.T) {
		html := renderLinkIconHTML(dir, "https://example.com/favicon.svg", "favicon", "")
		if !strings.Contains(html, `<object data="https://example.com/favicon.svg"`) {
			t.Fatalf("expected object tag: %s", html)
		}
	})
}

func TestFindExistingIconFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ih.apps.moo.teratan.net.svg"), []byte("svg"), 0644); err != nil {
		t.Fatal(err)
	}

	if got := findExistingIconFile(dir, "ih.apps.moo.teratan.net"); got != "ih.apps.moo.teratan.net.svg" {
		t.Fatalf("findExistingIconFile() = %q", got)
	}
}
