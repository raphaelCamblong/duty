// Package watch is duty's filesystem watcher: one fsnotify layer, shared by
// the TUI and the watch command. It watches every directory under a tree root
// and coalesces bursts of events into single notifications; callers re-read the
// tree on each notification.
package watch

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/raphaelCamblong/duty/internal/fsys"
)

const debounce = 100 * time.Millisecond

// Watcher re-walks the tree before each notification to watch directories
// that appeared, so new tracks refresh live too.
type Watcher struct {
	// C receives one value per debounced burst of events; it is closed when
	// the watcher stops.
	C     chan struct{}
	fsys  fsys.FS
	notif *fsnotify.Watcher
}

// Callers own Close.
func NewWatcher(filesystem fsys.FS, root string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("watch %s: %w", root, err)
	}
	if err := addDirs(filesystem, fw, root, true); err != nil {
		_ = fw.Close()
		return nil, err
	}
	watcher := &Watcher{C: make(chan struct{}, 1), fsys: filesystem, notif: fw}
	go watcher.loop(root)
	return watcher, nil
}

// Close stops the watcher; C is closed once the loop drains.
func (w *Watcher) Close() error {
	return w.notif.Close()
}

// loop drops a notification if the last one is still unread, since the
// re-scan is total anyway.
func (w *Watcher) loop(root string) {
	defer close(w.C)
	var fire <-chan time.Time
	for {
		select {
		case _, ok := <-w.notif.Events:
			if !ok {
				return
			}
			if fire == nil {
				fire = time.After(debounce)
			}
		case _, ok := <-w.notif.Errors:
			if !ok {
				return
			}
		case <-fire:
			fire = nil
			_ = addDirs(w.fsys, w.notif, root, false)
			select {
			case w.C <- struct{}{}:
			default:
			}
		}
	}
}

// Adding a watched path again is a no-op, so re-walks are cheap. With strict
// false, per-directory failures are skipped — directories vanish mid-walk
// when tasks are archived.
func addDirs(filesystem fsys.FS, fw *fsnotify.Watcher, root string, strict bool) error {
	return filesystem.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if strict {
				return fmt.Errorf("watch %s: %w", path, err)
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if err := fw.Add(path); err != nil && strict {
			return fmt.Errorf("watch %s: %w", path, err)
		}
		return nil
	})
}
