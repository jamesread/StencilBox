package watcher

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	log "github.com/sirupsen/logrus"
)

type DataFileWatcher struct {
	watcher       *fsnotify.Watcher
	buildConfigs  map[string]*buildconfigs.BuildConfig
	fileToConfigs map[string][]string    // Maps absolute file paths to config names
	debounceTimer map[string]*time.Timer // Debounce timer per config
	onRebuild     func(configName string)
	mu            sync.RWMutex
	debounceMu    sync.Mutex
}

// NewDataFileWatcher creates a new file system watcher for data files
func NewDataFileWatcher(buildConfigs map[string]*buildconfigs.BuildConfig, onRebuild func(configName string)) (*DataFileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dfw := &DataFileWatcher{
		watcher:       watcher,
		buildConfigs:  buildConfigs,
		fileToConfigs: make(map[string][]string),
		onRebuild:     onRebuild,
		debounceTimer: make(map[string]*time.Timer),
	}

	return dfw, nil
}

// Start begins watching all data files for the configured build configs
func (dfw *DataFileWatcher) Start(ctx context.Context) error {
	dfw.mu.Lock()
	defer dfw.mu.Unlock()

	// Add all data files to the watcher
	dfw.watchDatafiles(dfw.buildConfigs, "Watching datafile")

	// Start the event loop
	go dfw.watchLoop(ctx)

	log.Infof("Data file watcher started, monitoring %d files", len(dfw.fileToConfigs))
	return nil
}

// watchDatafiles adds the data files of every build config to the watcher. The caller holds dfw.mu.
func (dfw *DataFileWatcher) watchDatafiles(buildConfigs map[string]*buildconfigs.BuildConfig, logPrefix string) {
	for configName, config := range buildConfigs {
		dfw.watchConfigDatafiles(configName, config, logPrefix)
	}
}

func (dfw *DataFileWatcher) watchConfigDatafiles(configName string, config *buildconfigs.BuildConfig, logPrefix string) {
	configDir := filepath.Dir(config.Path)

	for datafileName, datafilePath := range config.Datafiles {
		// Construct absolute path
		absPath := filepath.Join(configDir, datafilePath)

		// Add the file to the watcher
		if err := dfw.watcher.Add(absPath); err != nil {
			log.Warnf("Failed to watch datafile %s for config %s: %v", absPath, configName, err)
			continue
		}

		log.Infof("%s: %s (name: %s) for config: %s", logPrefix, absPath, datafileName, configName)

		// Track which configs use this file
		dfw.fileToConfigs[absPath] = append(dfw.fileToConfigs[absPath], configName)
	}
}

// watchLoop processes file system events
func (dfw *DataFileWatcher) watchLoop(ctx context.Context) {
	for dfw.processNext(ctx) {
	}
}

// processNext handles the next watcher event, and reports whether to keep watching.
func (dfw *DataFileWatcher) processNext(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		log.Info("Data file watcher stopping...")
		_ = dfw.watcher.Close()
		return false

	case event, ok := <-dfw.watcher.Events:
		return dfw.onEvent(event, ok)

	case err, ok := <-dfw.watcher.Errors:
		return onWatcherError(err, ok)
	}
}

func (dfw *DataFileWatcher) onEvent(event fsnotify.Event, ok bool) bool {
	if !ok {
		return false
	}

	// We're interested in Write and Create events
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
		log.Infof("Data file modified: %s (operation: %s)", event.Name, event.Op)
		dfw.handleFileChange(event.Name)
	}

	return true
}

func onWatcherError(err error, ok bool) bool {
	if !ok {
		return false
	}

	log.Errorf("Watcher error: %v", err)

	return true
}

// handleFileChange processes a file change event with debouncing
func (dfw *DataFileWatcher) handleFileChange(filePath string) {
	dfw.mu.RLock()
	configNames, found := dfw.fileToConfigs[filePath]
	dfw.mu.RUnlock()

	if !found {
		log.Debugf("File %s is not associated with any build config", filePath)
		return
	}

	// Trigger rebuild for all configs that use this file
	for _, configName := range configNames {
		dfw.triggerRebuildWithDebounce(configName)
	}
}

// triggerRebuildWithDebounce triggers a rebuild after a short delay to avoid multiple rapid rebuilds
func (dfw *DataFileWatcher) triggerRebuildWithDebounce(configName string) {
	dfw.debounceMu.Lock()
	defer dfw.debounceMu.Unlock()

	// Cancel existing timer if any
	if timer, exists := dfw.debounceTimer[configName]; exists {
		timer.Stop()
	}

	// Create new timer with 1 second delay
	dfw.debounceTimer[configName] = time.AfterFunc(1*time.Second, func() {
		log.Infof("Triggering automatic rebuild for config: %s", configName)
		if dfw.onRebuild != nil {
			dfw.onRebuild(configName)
		}
	})
}

// UpdateBuildConfigs updates the watched files when build configs change
func (dfw *DataFileWatcher) UpdateBuildConfigs(buildConfigs map[string]*buildconfigs.BuildConfig) error {
	dfw.mu.Lock()
	defer dfw.mu.Unlock()

	// Remove all current watches
	for filePath := range dfw.fileToConfigs {
		_ = dfw.watcher.Remove(filePath)
	}

	// Clear the mapping
	dfw.fileToConfigs = make(map[string][]string)
	dfw.buildConfigs = buildConfigs

	// Re-add all data files
	dfw.watchDatafiles(buildConfigs, "Updated watch for datafile")

	return nil
}

// Stop stops the watcher
func (dfw *DataFileWatcher) Stop() {
	if dfw.watcher != nil {
		_ = dfw.watcher.Close()
	}
}
