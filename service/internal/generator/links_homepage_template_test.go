package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestLinksHomepageTemplateRendersWebuiUrl(t *testing.T) {
	t.Parallel()

	templatePath := filepath.Join("..", "..", "..", "templates", "links-homepage", "index.html")
	contents, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read template: %v", err)
	}

	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"linkIconHTML": func(iconPath, cssClass, extraAttrs string) string { return "" },
	}).Option("missingkey=error").Parse(string(contents))
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	data := map[string]any{
		"links": map[string]any{
			"title":      "Test homepage",
			"categories": []any{},
		},
		"hooks":              map[string]string{"head": "", "body": ""},
		"buildDateFormatted": "January 1, 2026 at 12:00 PM UTC",
		"version":            "test",
		"webuiUrl":           "/webui/build-config/My%20links%20homepage",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, `href="/webui/build-config/My%20links%20homepage"`) {
		t.Fatalf("rendered HTML missing webuiUrl href:\n%s", html)
	}
	if !strings.Contains(html, "StencilBox") {
		t.Fatal("rendered HTML missing StencilBox footer link text")
	}
}
