package clientapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesread/StencilBox/internal/generator"
)

func TestBuildHistoryPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, buildHistoryFilename)

	api := &ClientApi{
		buildHistory: loadBuildHistory(path),
		historyPath:  path,
	}

	api.recordBuildHistory("site", &generator.BuildStatus{Message: "ok"}, "site", true, 0)

	reloaded := loadBuildHistory(path)
	entries := reloaded["site"]

	if len(entries) != 1 || entries[0].Status != "ok" || !entries[0].IsAutoRebuild {
		t.Fatalf("unexpected reloaded history: %+v", reloaded)
	}
}

func TestLoadBuildHistoryCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), buildHistoryFilename)
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	if got := loadBuildHistory(path); len(got) != 0 {
		t.Fatalf("expected empty history, got %+v", got)
	}
}

func TestBuildHistoryPathOutsideOutputDir(t *testing.T) {
	t.Setenv("STENCILBOX_STATE_DIR", "")

	if got := buildHistoryPath("/config/output"); got != "/config/"+buildHistoryFilename {
		t.Fatalf("unexpected path %s", got)
	}

	t.Setenv("STENCILBOX_STATE_DIR", "/state")

	if got := buildHistoryPath("/config/output"); got != "/state/"+buildHistoryFilename {
		t.Fatalf("unexpected path %s", got)
	}
}
