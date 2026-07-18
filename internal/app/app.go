// Package app implements duty's use-cases: one method per verb, all
// orchestration over an injected fsys.FS. The sync invariant lives here —
// every mutating use-case edits the task file AND its board row in one call.
// Methods return data and errors; they never print and never parse flags.
package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// App bundles duty's use-cases over one filesystem.
type App struct {
	fs  fsys.FS
	now func() time.Time
}

// New returns an App operating on f, dating reports with the real clock.
func New(f fsys.FS) App {
	return NewWithClock(f, time.Now)
}

// NewWithClock returns an App like New, but reading the current time from now
// instead of the real clock — the seam tests use to fix report timestamps.
func NewWithClock(f fsys.FS, now func() time.Time) App {
	return App{fs: f, now: now}
}

// nameRE validates track folder names.
var nameRE = regexp.MustCompile(`^[a-z0-9-]+$`)

// unknownStatusErr is the one-line error every use-case rejecting an unknown
// status string returns.
func unknownStatusErr(status string) error {
	return fmt.Errorf("unknown status %q: want todo, in-progress, done, blocked or backlog", status)
}

// resolveOpenWithRoot resolves id to its open task file and the root of the
// tree containing cwd. Archived ids fail with tree.ErrArchived in the chain:
// archived tasks are read-only.
func (a App) resolveOpenWithRoot(cwd, id string) (root, path string, err error) {
	root, err = tree.FindRoot(a.fs, cwd)
	if err != nil {
		return "", "", err
	}
	path, err = tree.ResolveTask(a.fs, root, id)
	if err != nil {
		return "", "", err
	}
	return root, path, nil
}

// lock takes the tree-wide write lock for the tree at root and returns its
// release function. Every mutating use-case holds it for its whole duration so
// parallel writers serialize on the board rather than racing on a shared file.
func (a App) lock(root string) (func(), error) {
	return a.fs.Lock(filepath.Join(root, names.LockFile))
}

// lockTree finds cwd's tree root and takes the tree-wide write lock, returning
// its release function — for mutators whose root is needed only to lock.
func (a App) lockTree(cwd string) (func(), error) {
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	return a.lock(root)
}

// applyEdit reads the file at path, transforms its bytes through fn, and writes
// the result back — the read → transform → write spine of the single-file edits.
func (a App) applyEdit(path string, fn func([]byte) ([]byte, error)) error {
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := fn(content)
	if err != nil {
		return err
	}
	return a.fs.WriteFile(path, out)
}

// lockedEdit runs applyEdit under the tree write lock at root.
func (a App) lockedEdit(root, path string, fn func([]byte) ([]byte, error)) error {
	unlock, err := a.lock(root)
	if err != nil {
		return err
	}
	defer unlock()
	return a.applyEdit(path, fn)
}

// walkBoards returns the board an --in-scoped read targets and every board
// below it — the contextBoard→Boards prelude the multi-board reads share.
func (a App) walkBoards(cwd, in string) (boardDir string, boards []string, err error) {
	boardDir, err = a.contextBoard(cwd, in)
	if err != nil {
		return "", nil, err
	}
	boards, err = tree.Boards(a.fs, boardDir)
	if err != nil {
		return "", nil, err
	}
	return boardDir, boards, nil
}

// contextBoard returns the board an --in-scoped command targets: the board
// containing cwd when in is empty (the cwd walk-up default), else the board at
// the root-relative slash path in ("." = root board), validated to exist.
func (a App) contextBoard(cwd, in string) (string, error) {
	if in == "" {
		return tree.CurrentBoard(a.fs, cwd)
	}
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return "", err
	}
	return tree.ResolveTrack(a.fs, root, in)
}

// boardIndexPath returns the path of the board index in dir.
func boardIndexPath(dir string) string {
	return filepath.Join(dir, names.BoardFile)
}

// boardBeside returns the path of the board index in the same directory as
// the task file at taskPath.
func boardBeside(taskPath string) string {
	return boardIndexPath(filepath.Dir(taskPath))
}

// readTask reads and parses the task file at path, returning the parsed task
// and the raw contents. Read errors pass through unwrapped so callers can
// branch on fs.ErrNotExist; parse errors carry the path.
func (a App) readTask(path string) (task.Task, []byte, error) {
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return task.Task{}, nil, err
	}
	t, err := parseTask(path, content)
	if err != nil {
		return task.Task{}, nil, err
	}
	return t, content, nil
}

// parseTask parses a task file's raw content, wrapping a parse error with path.
func parseTask(path string, content []byte) (task.Task, error) {
	t, err := task.Parse(content)
	if err != nil {
		return task.Task{}, fmt.Errorf("%s: %w", path, err)
	}
	return t, nil
}

// tasksIn reads and parses every task file directly in dir, returning the
// filenames and parsed tasks in matching board-directory order.
func (a App) tasksIn(dir string) (files []string, tasks []task.Task, err error) {
	files, err = tree.TaskFileNames(a.fs, dir)
	if err != nil {
		return nil, nil, err
	}
	tasks = make([]task.Task, 0, len(files))
	for _, name := range files {
		t, _, err := a.readTask(filepath.Join(dir, name))
		if err != nil {
			return nil, nil, err
		}
		tasks = append(tasks, t)
	}
	return files, tasks, nil
}
