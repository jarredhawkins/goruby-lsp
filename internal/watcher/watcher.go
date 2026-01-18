package watcher

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// ChangeHandler is called when files change
type ChangeHandler func(changed, removed []string)

// Watcher monitors Ruby files for changes using fsnotify
type Watcher struct {
	watcher   *fsnotify.Watcher
	rootPath  string
	handler   ChangeHandler
	debouncer *Debouncer
	done      chan struct{}
}

// New creates a new file watcher for the root path
func New(rootPath string, handler ChangeHandler) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:   fsw,
		rootPath:  rootPath,
		handler:   handler,
		debouncer: NewDebouncer(100), // 100ms debounce
		done:      make(chan struct{}),
	}

	return w, nil
}

// Start begins watching for file changes
func (w *Watcher) Start() error {
	// Add all directories recursively
	err := filepath.WalkDir(w.rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if d.IsDir() {
			name := d.Name()
			// Skip hidden and vendor directories
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}

			if err := w.watcher.Add(path); err != nil {
				log.Printf("failed to watch %s: %v", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Start the event loop
	go w.eventLoop()

	log.Printf("file watcher started for %s", w.rootPath)
	return nil
}

func (w *Watcher) eventLoop() {
	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name

	// Check if it's a directory event
	if event.Has(fsnotify.Create) {
		// If a new directory was created, watch it
		if info, err := os.Lstat(path); err == nil && info.IsDir() {
			name := filepath.Base(path)
			if !strings.HasPrefix(name, ".") && name != "vendor" && name != "node_modules" {
				if err := w.watcher.Add(path); err != nil {
					log.Printf("failed to watch new directory %s: %v", path, err)
				}
			}
			return
		}
	}

	// Only process Ruby files
	if !isRubyFile(path) {
		return
	}

	// Debounce and dispatch changes
	w.debouncer.Add(path, event.Op)
	w.debouncer.Flush(func(changed, removed []string) {
		if len(changed) > 0 || len(removed) > 0 {
			log.Printf("file changes: %d changed, %d removed", len(changed), len(removed))
			w.handler(changed, removed)
		}
	})
}

// Close stops the watcher
func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}

// isRubyFile checks if a file is a Ruby file
func isRubyFile(path string) bool {
	ext := filepath.Ext(path)
	base := filepath.Base(path)

	switch ext {
	case ".rb", ".rake", ".gemspec":
		return true
	}

	switch base {
	case "Gemfile", "Rakefile", "Guardfile", "Vagrantfile":
		return true
	}

	return false
}
