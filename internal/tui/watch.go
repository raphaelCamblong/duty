package tui

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/raphaelCamblong/duty/internal/fsys"
)

// debounce is how long the watcher coalesces a burst of filesystem events
// into one refresh notification.
const debounce = 100 * time.Millisecond

// Watcher watches every directory under a tree root and coalesces bursts of
// filesystem events into single notifications on C. Before each notification
// it re-walks the tree to watch directories that appeared, so new sub-boards
// refresh live too.
type Watcher struct {
	// C receives one value per debounced burst of events; it is closed when
	// the watcher stops.
	C     chan struct{}
	fsys  fsys.FS
	notif *fsnotify.Watcher
}

// NewWatcher watches every directory under root and starts the debounce
// loop. Callers own Close.
func NewWatcher(f fsys.FS, root string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("watch %s: %w", root, err)
	}
	if err := addDirs(f, fw, root, true); err != nil {
		fw.Close()
		return nil, err
	}
	w := &Watcher{C: make(chan struct{}, 1), fsys: f, notif: fw}
	go w.loop(root)
	return w, nil
}

// Close stops the watcher; C is closed once the loop drains.
func (w *Watcher) Close() error {
	return w.notif.Close()
}

// loop debounces events: the first event of a burst arms a timer; when it
// fires, the tree is re-walked for new directories and one notification is
// sent (dropped if the last one is still unread — the re-scan is total
// anyway).
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
			addDirs(w.fsys, w.notif, root, false)
			select {
			case w.C <- struct{}{}:
			default:
			}
		}
	}
}

// addDirs walks root through the port and watches every directory. Adding a
// watched path again is a no-op, so re-walks are cheap. With strict false,
// per-directory failures are skipped — directories vanish mid-walk when tasks
// are archived.
func addDirs(f fsys.FS, fw *fsnotify.Watcher, root string, strict bool) error {
	return f.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if strict {
				return fmt.Errorf("watch %s: %w", path, err)
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if err := fw.Add(path); err != nil && strict {
			return fmt.Errorf("watch %s: %w", path, err)
		}
		return nil
	})
}
