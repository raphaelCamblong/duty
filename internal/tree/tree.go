// Package tree locates a duty tree on the filesystem: its root, its boards,
// its task files, and the next task number. It reads directory structure and
// filenames only — file contents are owned by the task and board packages.
package tree

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ErrArchived reports that an id resolved to a task inside an archive/
// directory; archived tasks are read-only by convention.
var ErrArchived = errors.New("archived tasks are read-only")

// BoardFile is the marker file that makes a directory a board. Every other
// package that needs the board index's filename imports this constant
// instead of repeating the literal.
const BoardFile = "BOARD.md"

// ConfigFile is the project config file; its presence marks the tree root.
const ConfigFile = "duty.toml"

// ArchiveDir is the name of a board's completed-tasks subdirectory.
const ArchiveDir = "archive"

// ReadmeFile is the convention doc `duty init` writes next to the root board.
const ReadmeFile = "README.md"

// taskNN extracts the numeric part of a task filename (T-NN-<slug>.md).
var taskNN = regexp.MustCompile(`^T-(\d+)-.*\.md$`)

// FindRoot returns the root of the duty tree containing cwd. It walks up to
// the nearest directory holding a BOARD.md, then keeps ascending while the
// parent also holds one; a directory holding duty.toml marks the root
// explicitly and stops the ascent. Outside a tree it falls back to ./duty/
// if that directory exists.
func FindRoot(cwd string) (string, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("find root: %w", err)
	}
	board, ok := nearestBoard(abs)
	if !ok {
		return fallbackTree(abs)
	}
	root := board
	for {
		if hasFile(root, ConfigFile) {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root || !hasFile(parent, BoardFile) {
			return root, nil
		}
		root = parent
	}
}

// CurrentBoard returns the nearest ancestor of cwd (including cwd itself)
// holding a BOARD.md. Outside a tree it falls back to ./duty/ if that
// directory exists.
func CurrentBoard(cwd string) (string, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("current board: %w", err)
	}
	if board, ok := nearestBoard(abs); ok {
		return board, nil
	}
	return fallbackTree(abs)
}

// Boards walks the tree under root and returns every directory holding a
// BOARD.md, in lexical order, skipping archive/ directories. A duty.toml
// anywhere below root is an error: it would declare a second root.
func Boards(root string) ([]string, error) {
	var boards []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("scan boards: %w", err)
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == ArchiveDir && path != root {
			return fs.SkipDir
		}
		if path != root && hasFile(path, ConfigFile) {
			return fmt.Errorf("second duty.toml found in %s: only the tree root may hold one", path)
		}
		if hasFile(path, BoardFile) {
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
func ResolveTask(root, id string) (string, error) {
	prefix := id + "-"
	var found string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		return "", fmt.Errorf("task %s not found", id)
	}
	if underArchive(root, found) {
		return "", fmt.Errorf("task %s is archived: %w", id, ErrArchived)
	}
	return found, nil
}

// NextNN walks every task filename under root — open and archived, every
// board — and returns the next task number, zero-padded to two digits
// minimum.
func NextNN(root string) (string, error) {
	highest := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
// BOARD.md, or false if the walk reaches the filesystem root without one.
func nearestBoard(dir string) (string, bool) {
	for {
		if hasFile(dir, BoardFile) {
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
func fallbackTree(cwd string) (string, error) {
	fallback := filepath.Join(cwd, "duty")
	info, err := os.Stat(fallback)
	if err == nil && info.IsDir() {
		return fallback, nil
	}
	return "", fmt.Errorf("no duty tree found (no BOARD.md above %s and no ./duty)", cwd)
}

// hasFile reports whether dir contains a non-directory entry named name.
func hasFile(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
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
		if part == ArchiveDir {
			return true
		}
	}
	return false
}
