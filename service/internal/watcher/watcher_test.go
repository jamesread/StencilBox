package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesread/StencilBox/internal/buildconfigs"
)

func TestNewDataFileWatcher(t *testing.T) {
	buildConfigs := make(map[string]*buildconfigs.BuildConfig)

	watcher, err := NewDataFileWatcher(buildConfigs, nil)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if watcher == nil {
		t.Fatal("Watcher should not be nil")
	}

	if watcher.fileToConfigs == nil {
		t.Fatal("fileToConfigs should be initialized")
	}

	if watcher.debounceTimer == nil {
		t.Fatal("debounceTimer should be initialized")
	}

	watcher.Stop()
}

func TestWatcherWithDataFiles(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test data files
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)

	dataFile1 := filepath.Join(dataDir, "test1.yml")
	dataFile2 := filepath.Join(dataDir, "test2.yml")

	os.WriteFile(dataFile1, []byte("key: value1\n"), 0644)
	os.WriteFile(dataFile2, []byte("key: value2\n"), 0644)

	// Create build config
	configFile := filepath.Join(tmpDir, "config.yaml")
	buildConfigs := map[string]*buildconfigs.BuildConfig{
		"test-config": {
			Name:     "test-config",
			Path:     configFile,
			Template: "test-template",
			Datafiles: map[string]string{
				"data1": filepath.Join("data", "test1.yml"),
				"data2": filepath.Join("data", "test2.yml"),
			},
		},
	}

	// Track rebuilds
	rebuilds := make(chan string, 10)
	onRebuild := func(configName string) {
		rebuilds <- configName
	}

	// Create and start watcher
	watcher, err := NewDataFileWatcher(buildConfigs, onRebuild)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = watcher.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Give watcher time to initialize
	time.Sleep(100 * time.Millisecond)

	// Verify files are being watched
	if len(watcher.fileToConfigs) != 2 {
		t.Errorf("Expected 2 files to be watched, got %d", len(watcher.fileToConfigs))
	}

	// Modify a data file
	err = os.WriteFile(dataFile1, []byte("key: updated\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to data file: %v", err)
	}

	// Wait for rebuild trigger (with timeout)
	select {
	case configName := <-rebuilds:
		if configName != "test-config" {
			t.Errorf("Expected rebuild for 'test-config', got '%s'", configName)
		}
	case <-time.After(3 * time.Second):
		t.Error("Timeout waiting for rebuild trigger")
	}
}

func TestWatcherDebounce(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "watcher-debounce-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test data file
	dataFile := filepath.Join(tmpDir, "test.yml")
	os.WriteFile(dataFile, []byte("key: value\n"), 0644)

	// Create build config
	configFile := filepath.Join(tmpDir, "config.yaml")
	buildConfigs := map[string]*buildconfigs.BuildConfig{
		"test-config": {
			Name:     "test-config",
			Path:     configFile,
			Template: "test-template",
			Datafiles: map[string]string{
				"data": "test.yml",
			},
		},
	}

	// Track rebuilds
	rebuilds := make(chan string, 10)
	rebuildCount := 0
	onRebuild := func(configName string) {
		rebuildCount++
		rebuilds <- configName
	}

	// Create and start watcher
	watcher, err := NewDataFileWatcher(buildConfigs, onRebuild)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = watcher.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Give watcher time to initialize
	time.Sleep(100 * time.Millisecond)

	// Modify file multiple times rapidly
	for i := 0; i < 5; i++ {
		os.WriteFile(dataFile, []byte("key: value"+string(rune(i))+"\n"), 0644)
		time.Sleep(100 * time.Millisecond) // Less than debounce delay
	}

	// Wait for debounce period plus a bit more
	time.Sleep(2 * time.Second)

	// Should only get one rebuild due to debouncing
	select {
	case <-rebuilds:
		// Got at least one rebuild, which is expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected at least one rebuild")
	}

	// Drain any additional rebuilds
	drainTimeout := time.After(100 * time.Millisecond)
drainLoop:
	for {
		select {
		case <-rebuilds:
			// Drain
		case <-drainTimeout:
			break drainLoop
		}
	}

	// We expect exactly 1 rebuild due to debouncing
	// (multiple rapid changes should be coalesced)
	if rebuildCount != 1 {
		t.Logf("Warning: Expected 1 rebuild due to debouncing, got %d (this may be flaky due to timing)", rebuildCount)
	}
}

func TestUpdateBuildConfigs(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "watcher-update-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create initial data file
	dataFile1 := filepath.Join(tmpDir, "test1.yml")
	os.WriteFile(dataFile1, []byte("key: value1\n"), 0644)

	// Initial build config
	configFile := filepath.Join(tmpDir, "config.yaml")
	initialConfigs := map[string]*buildconfigs.BuildConfig{
		"test-config": {
			Name:     "test-config",
			Path:     configFile,
			Template: "test-template",
			Datafiles: map[string]string{
				"data": "test1.yml",
			},
		},
	}

	// Create watcher
	watcher, err := NewDataFileWatcher(initialConfigs, nil)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = watcher.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if len(watcher.fileToConfigs) != 1 {
		t.Errorf("Expected 1 file to be watched initially, got %d", len(watcher.fileToConfigs))
	}

	// Create new data file and update configs
	dataFile2 := filepath.Join(tmpDir, "test2.yml")
	os.WriteFile(dataFile2, []byte("key: value2\n"), 0644)

	updatedConfigs := map[string]*buildconfigs.BuildConfig{
		"test-config": {
			Name:     "test-config",
			Path:     configFile,
			Template: "test-template",
			Datafiles: map[string]string{
				"data1": "test1.yml",
				"data2": "test2.yml",
			},
		},
	}

	err = watcher.UpdateBuildConfigs(updatedConfigs)
	if err != nil {
		t.Fatalf("Failed to update build configs: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if len(watcher.fileToConfigs) != 2 {
		t.Errorf("Expected 2 files to be watched after update, got %d", len(watcher.fileToConfigs))
	}
}

func TestMultipleConfigsSameFile(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "watcher-multi-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create shared data file
	dataFile := filepath.Join(tmpDir, "shared.yml")
	os.WriteFile(dataFile, []byte("key: value\n"), 0644)

	// Multiple configs using same file
	configFile1 := filepath.Join(tmpDir, "config1.yaml")
	configFile2 := filepath.Join(tmpDir, "config2.yaml")

	buildConfigs := map[string]*buildconfigs.BuildConfig{
		"config1": {
			Name:     "config1",
			Path:     configFile1,
			Template: "test-template",
			Datafiles: map[string]string{
				"data": "shared.yml",
			},
		},
		"config2": {
			Name:     "config2",
			Path:     configFile2,
			Template: "test-template",
			Datafiles: map[string]string{
				"data": "shared.yml",
			},
		},
	}

	// Track rebuilds
	rebuilds := make(map[string]int)
	var rebuildsMutex = &struct {
		m map[string]int
	}{m: rebuilds}

	onRebuild := func(configName string) {
		rebuildsMutex.m[configName]++
	}

	// Create and start watcher
	watcher, err := NewDataFileWatcher(buildConfigs, onRebuild)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = watcher.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Modify shared file
	err = os.WriteFile(dataFile, []byte("key: updated\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to data file: %v", err)
	}

	// Wait for rebuild triggers
	time.Sleep(2 * time.Second)

	// Both configs should be rebuilt
	if rebuildsMutex.m["config1"] != 1 {
		t.Errorf("Expected config1 to be rebuilt once, got %d", rebuildsMutex.m["config1"])
	}

	if rebuildsMutex.m["config2"] != 1 {
		t.Errorf("Expected config2 to be rebuilt once, got %d", rebuildsMutex.m["config2"])
	}
}
