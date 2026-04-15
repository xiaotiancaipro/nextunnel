package utils

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

const (
	debouncedReloadDelay     = 400 * time.Millisecond
	configReloadPollInterval = 2 * time.Second
)

// WatchConfigChanges calls reload when the config file at path changes (debounced).
// Falls back to polling if fsnotify is unavailable or the directory cannot be watched.
func WatchConfigChanges(ctx context.Context, path string, reload func(source string), logger *logrus.Logger) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Warnf("config watch: abs path %s: %v, using polling", path, err)
		watchConfigChangesPoll(ctx, path, reload, logger)
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Warnf("config watch: fsnotify: %v, using polling", err)
		watchConfigChangesPoll(ctx, path, reload, logger)
		return
	}
	defer watcher.Close()

	dir := filepath.Dir(absPath)
	if err := watcher.Add(dir); err != nil {
		logger.Warnf("config watch: add %s: %v, using polling", dir, err)
		watchConfigChangesPoll(ctx, path, reload, logger)
		return
	}

	var mu sync.Mutex
	var debounce *time.Timer

	schedule := func() {
		mu.Lock()
		defer mu.Unlock()
		if debounce != nil {
			debounce.Stop()
		}
		debounce = time.AfterFunc(debouncedReloadDelay, func() { reload("config file") })
	}

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			if debounce != nil {
				debounce.Stop()
			}
			mu.Unlock()
			return
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if !configPathMatches(absPath, ev.Name) {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove|fsnotify.Chmod) == 0 {
				continue
			}
			schedule()
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Warnf("config watch error: %v", err)
		}
	}
}

func configPathMatches(absConfig, eventPath string) bool {
	eventPath = filepath.Clean(eventPath)
	absConfig = filepath.Clean(absConfig)
	if eventPath == absConfig {
		return true
	}
	return filepath.Base(eventPath) == filepath.Base(absConfig) &&
		filepath.Dir(eventPath) == filepath.Dir(absConfig)
}

func watchConfigChangesPoll(ctx context.Context, path string, reload func(source string), logger *logrus.Logger) {
	ticker := time.NewTicker(configReloadPollInterval)
	defer ticker.Stop()

	var lastMod time.Time
	if fi, err := os.Stat(path); err == nil {
		lastMod = fi.ModTime()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fi, err := os.Stat(path)
			if err != nil {
				logger.Warnf("config watch: stat %s: %v", path, err)
				continue
			}
			mod := fi.ModTime()
			if mod.Equal(lastMod) {
				continue
			}
			lastMod = mod
			reload("config file")
		}
	}
}
