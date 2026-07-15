// Package tree locates a duty tree on the filesystem: its root, its boards,
// its task files, and the next task number. It reads directory structure and
// filenames only — file contents are owned by the task and board packages.
package tree

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
)

// ErrArchived reports that an id resolved to a task inside an archive/
// directory; archived tasks are read-only by convention.
var ErrArchived = errors.New("archived tasks are read-only")

// taskNN extracts the numeric part of a task filename (T-NN-<slug>.md).
var taskNN = regexp.MustCompile(`^` + regexp.QuoteMeta(task.IDPrefix) + `(\d+)-.*\.md$`)

// FindRoot returns the root of the duty tree containing cwd. It walks up to
// the nearest directory holding a board index, then keeps ascending while the
// parent also holds one; a directory holding the config file marks the root
// explicitly and stops the ascent. Outside a tree it falls back to ./duty/
// if that directory exists.
func FindRoot(f fsys.FS, cwd string) (string, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("find root: %w", err)
	}
	board, ok := nearestBoard(f, abs)
	if !ok {
		return fallbackTree(f, abs)
	}
	root := board
	for {
		if hasFile(f, root, names.ConfigFile) {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root || !hasFile(f, parent, names.BoardFile) {
			return root, nil
		}
		root = parent
	}
}

// CurrentBoard returns the nearest ancestor of cwd (including cwd itself)
// holding a board index. Outside a tree it falls back to ./duty/ if that
// directory exists.
func CurrentBoard(f fsys.FS, cwd string) (string, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("current board: %w", err)
	}
	if board, ok := nearestBoard(f, abs); ok {
		return board, nil
	}
	return fallbackTree(f, abs)
}

// Boards walks the tree under root and returns every directory holding a board
// index, in lexical order, skipping archive/ directories. A config file
// anywhere below root is an error: it would declare a second root.
func Boards(f fsys.FS, root string) ([]string, error) {
	var boards []string
	err := f.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("scan boards: %w", err)
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == names.ArchiveDir && path != root {
			return fs.SkipDir
		}
		if path != root && hasFile(f, path, names.ConfigFile) {
			return fmt.Errorf("second %s found in %s: only the tree root may hold one", names.ConfigFile, path)
		}
		if hasFile(f, path, names.BoardFile) {
			boards = append(boards, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return boards, nil
}

// ResolveTask walks the tree under root for the task file named <id>-*.md
// and returns its path. A match inside an archive/ directory is an error
// wrapping ErrArchived: archived tasks are read-only.
func ResolveTask(f fsys.FS, root, id string) (string, error) {
	prefix := id + "-"
	var found string
	err := f.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("resolve %s: %w", id, err)
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".md") {
			return nil
		}
		found = path
		return fs.SkipAll
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("unknown task id %q — try 'duty get tasks'", id)
	}
	if underArchive(root, found) {
		return "", fmt.Errorf("task %s is archived: %w", id, ErrArchived)
	}
	return found, nil
}

// ResolveTrack resolves track — a slash path relative to root, "." naming the
// root board — to an existing board directory inside the tree. An absolute or
// escaping path, or a path naming no board, is the one failure: unknown track %q.
func ResolveTrack(f fsys.FS, root, track string) (string, error) {
	dir := filepath.Join(root, filepath.FromSlash(track))
	rel, err := filepath.Rel(root, dir)
	if err != nil || !filepath.IsLocal(rel) || !hasFile(f, dir, names.BoardFile) {
		return "", fmt.Errorf("unknown track %q", track)
	}
	return dir, nil
}

// IsTaskFile reports whether name is a task filename: T-NN-<slug>.md.
func IsTaskFile(name string) bool {
	return taskNN.MatchString(name)
}

// TaskFileNames returns the task filenames directly in dir, in ReadDir order,
// skipping subdirectories and non-task files.
func TaskFileNames(f fsys.FS, dir string) ([]string, error) {
	entries, err := f.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !IsTaskFile(e.Name()) {
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

// NextNN walks every task filename under root — open and archived, every
// board — and returns the next task number, zero-padded to two digits
// minimum.
func NextNN(f fsys.FS, root string) (string, error) {
	highest := 0
	err := f.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("scan task numbers: %w", err)
		}
		if d.IsDir() {
			return nil
		}
		m := taskNN.FindStringSubmatch(d.Name())
		if m == nil {
			return nil
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return nil
		}
		if n > highest {
			highest = n
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%02d", highest+1), nil
}

// nearestBoard walks up from dir and returns the first directory holding a
// board index, or false if the walk reaches the filesystem root without one.
func nearestBoard(f fsys.FS, dir string) (string, bool) {
	for {
		if hasFile(f, dir, names.BoardFile) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// fallbackTree returns cwd/duty if it is a directory, else a one-line error:
// the conventional tree location when cwd is outside any tree.
func fallbackTree(f fsys.FS, cwd string) (string, error) {
	fallback := filepath.Join(cwd, names.TreeDir)
	info, err := f.Stat(fallback)
	if err == nil && info.IsDir() {
		return fallback, nil
	}
	return "", fmt.Errorf("no duty tree found (no %s above %s and no ./%s)", names.BoardFile, cwd, names.TreeDir)
}

// hasFile reports whether dir contains a non-directory entry named name.
func hasFile(f fsys.FS, dir, name string) bool {
	info, err := f.Stat(filepath.Join(dir, name))
	return err == nil && !info.IsDir()
}

// underArchive reports whether path sits below an archive/ directory
// relative to root.
func underArchive(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == names.ArchiveDir {
			return true
		}
	}
	return false
}
