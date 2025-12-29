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
	fileToConfigs map[string][]string // Maps absolute file paths to config names
	mu            sync.RWMutex
	onRebuild     func(configName string)
	debounceTimer map[string]*time.Timer // Debounce timer per config
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
	for configName, config := range dfw.buildConfigs {
		configDir := filepath.Dir(config.Path)

		for datafileName, datafilePath := range config.Datafiles {
			// Construct absolute path
			absPath := filepath.Join(configDir, datafilePath)

			// Add the file to the watcher
			err := dfw.watcher.Add(absPath)
			if err != nil {
				log.Warnf("Failed to watch datafile %s for config %s: %v", absPath, configName, err)
				continue
			}

			log.Infof("Watching datafile: %s (name: %s) for config: %s", absPath, datafileName, configName)

			// Track which configs use this file
			if dfw.fileToConfigs[absPath] == nil {
				dfw.fileToConfigs[absPath] = make([]string, 0)
			}
			dfw.fileToConfigs[absPath] = append(dfw.fileToConfigs[absPath], configName)
		}
	}

	// Start the event loop
	go dfw.watchLoop(ctx)

	log.Infof("Data file watcher started, monitoring %d files", len(dfw.fileToConfigs))
	return nil
}

// watchLoop processes file system events
func (dfw *DataFileWatcher) watchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Info("Data file watcher stopping...")
			dfw.watcher.Close()
			return

		case event, ok := <-dfw.watcher.Events:
			if !ok {
				return
			}

			// We're interested in Write and Create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Infof("Data file modified: %s (operation: %s)", event.Name, event.Op)
				dfw.handleFileChange(event.Name)
			}

		case err, ok := <-dfw.watcher.Errors:
			if !ok {
				return
			}
			log.Errorf("Watcher error: %v", err)
		}
	}
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
		dfw.watcher.Remove(filePath)
	}

	// Clear the mapping
	dfw.fileToConfigs = make(map[string][]string)
	dfw.buildConfigs = buildConfigs

	// Re-add all data files
	for configName, config := range buildConfigs {
		configDir := filepath.Dir(config.Path)

		for datafileName, datafilePath := range config.Datafiles {
			absPath := filepath.Join(configDir, datafilePath)

			err := dfw.watcher.Add(absPath)
			if err != nil {
				log.Warnf("Failed to watch datafile %s for config %s: %v", absPath, configName, err)
				continue
			}

			log.Infof("Updated watch for datafile: %s (name: %s) for config: %s", absPath, datafileName, configName)

			if dfw.fileToConfigs[absPath] == nil {
				dfw.fileToConfigs[absPath] = make([]string, 0)
			}
			dfw.fileToConfigs[absPath] = append(dfw.fileToConfigs[absPath], configName)
		}
	}

	return nil
}

// Stop stops the watcher
func (dfw *DataFileWatcher) Stop() {
	if dfw.watcher != nil {
		dfw.watcher.Close()
	}
}
