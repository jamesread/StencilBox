package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemplate(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name, "index.html"), []byte(name), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFindTemplatePrefersCustom(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("STENCILBOX_TEMPLATES_DIR", custom)

	// Tests run from internal/generator, so the built-in templates are not on the search path;
	// only the custom directory is found.
	writeTemplate(t, custom, "mine")

	got, err := FindTemplate("mine")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != TemplateSourceCustom || got.Path != filepath.Join(custom, "mine") {
		t.Fatalf("unexpected template dir %+v", got)
	}

	if CustomTemplatesDir() != custom {
		t.Fatalf("expected custom templates dir %s, got %s", custom, CustomTemplatesDir())
	}
}

func TestFindTemplateRejectsBadNames(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("STENCILBOX_TEMPLATES_DIR", custom)
	writeTemplate(t, filepath.Dir(custom), filepath.Base(custom)+"-outside")

	for _, name := range []string{"", "..", "../" + filepath.Base(custom) + "-outside", "missing"} {
		if _, err := FindTemplate(name); err == nil {
			t.Errorf("expected error for template name %q", name)
		}
	}
}
