package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesread/StencilBox/internal/buildconfigs"
)

func TestBuildRepoHooksCombinesAllRepos(t *testing.T) {
	dir := t.TempDir()
	cfg := &buildconfigs.BuildConfig{
		Path: filepath.Join(dir, "site.yaml"),
		Repos: []buildconfigs.GitRepo{
			{URL: "https://example.com/one.git"},
			{URL: "https://example.com/two.git"},
		},
	}

	for _, repo := range []string{"one", "two"} {
		repoDir := filepath.Join(dir, "repos", repo)
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repoDir, "head.html"), []byte("<!-- "+repo+" -->"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	updates := make(chan string, 10)
	head := (*buildRepoHooks(cfg, updates))["head"]

	if !strings.Contains(head, "<!-- one -->") || !strings.Contains(head, "<!-- two -->") {
		t.Fatalf("expected head hooks from both repos, got %q", head)
	}
}

func TestGenerateStreamsFailureReason(t *testing.T) {
	dir := t.TempDir()
	cfg := &buildconfigs.BuildConfig{
		Name:      "broken",
		OutputDir: "broken",
		Template:  "debug",
		Path:      filepath.Join(dir, "broken.yaml"),
		Datafiles: map[string]string{"links": "missing.yaml"},
	}

	status := &BuildStatus{}
	updates := make(chan string)
	go Generate(nil, filepath.Join(dir, "out"), cfg, status, updates)

	var last string
	for u := range updates {
		last = u
	}

	if !status.IsError {
		t.Fatal("expected build to fail")
	}
	if status.Message == "" || last != status.Message {
		t.Fatalf("expected failure reason %q to be the last streamed update, got %q", status.Message, last)
	}
}
