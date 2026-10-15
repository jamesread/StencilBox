package clientapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	log "github.com/sirupsen/logrus"
)

const buildHistoryFilename = "stencilbox-build-history.json"

// buildHistoryPath returns where build history is persisted. It deliberately sits outside
// the output directory, because everything in there is served publicly.
func buildHistoryPath(baseOutputDir string) string {
	if dir := strings.TrimSpace(os.Getenv("STENCILBOX_STATE_DIR")); dir != "" {
		return filepath.Join(dir, buildHistoryFilename)
	}

	return filepath.Join(filepath.Dir(baseOutputDir), buildHistoryFilename)
}

// loadBuildHistory reads persisted build history. A missing or unreadable file yields empty history.
func loadBuildHistory(path string) map[string][]*pb.BuildHistoryEntry {
	history := make(map[string][]*pb.BuildHistoryEntry)

	content, err := os.ReadFile(path)

	if os.IsNotExist(err) {
		return history
	}

	if err != nil {
		log.Warnf("Failed to read build history from %s: %v", path, err)
		return history
	}

	if err := json.Unmarshal(content, &history); err != nil {
		log.Warnf("Failed to parse build history from %s, starting with empty history: %v", path, err)
		return make(map[string][]*pb.BuildHistoryEntry)
	}

	log.Infof("Loaded build history for %d build configs from %s", len(history), path)

	return history
}

// saveBuildHistory writes build history atomically, so a crash mid-write cannot corrupt it.
func saveBuildHistory(path string, history map[string][]*pb.BuildHistoryEntry) error {
	content, err := json.Marshal(history)
	if err != nil {
		return err
	}

	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, content, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}
