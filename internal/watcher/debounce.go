package watcher

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// pendingChange tracks a file change event
type pendingChange struct {
	path      string
	op        fsnotify.Op
	timestamp time.Time
}

// Debouncer batches file change events to avoid redundant processing
type Debouncer struct {
	mu       sync.Mutex
	pending  map[string]*pendingChange
	interval time.Duration
	timer    *time.Timer
}

// NewDebouncer creates a new debouncer with the given interval in milliseconds
func NewDebouncer(intervalMs int) *Debouncer {
	return &Debouncer{
		pending:  make(map[string]*pendingChange),
		interval: time.Duration(intervalMs) * time.Millisecond,
	}
}

// Add records a file change event
func (d *Debouncer) Add(path string, op fsnotify.Op) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if existing, ok := d.pending[path]; ok {
		// Combine operations
		existing.op |= op
		existing.timestamp = time.Now()
	} else {
		d.pending[path] = &pendingChange{
			path:      path,
			op:        op,
			timestamp: time.Now(),
		}
	}
}

// Flush processes pending changes after the debounce interval
func (d *Debouncer) Flush(callback func(changed, removed []string)) {
	d.mu.Lock()

	// Cancel any existing timer
	if d.timer != nil {
		d.timer.Stop()
	}

	// Set a new timer
	d.timer = time.AfterFunc(d.interval, func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		if len(d.pending) == 0 {
			return
		}

		var changed, removed []string

		for path, change := range d.pending {
			if change.op.Has(fsnotify.Remove) || change.op.Has(fsnotify.Rename) {
				removed = append(removed, path)
			} else if change.op.Has(fsnotify.Write) || change.op.Has(fsnotify.Create) {
				changed = append(changed, path)
			}
		}

		// Clear pending changes
		d.pending = make(map[string]*pendingChange)

		// Call the callback outside the lock
		if len(changed) > 0 || len(removed) > 0 {
			go callback(changed, removed)
		}
	})

	d.mu.Unlock()
}
