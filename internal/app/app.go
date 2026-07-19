// Package app implements duty's use-cases: one method per verb, all
// orchestration over an injected fsys.FS. The sync invariant lives here —
// every mutating use-case edits the task file AND its board row in one call.
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

type App struct {
	fs  fsys.FS
	now func() time.Time
}

func New(fs fsys.FS) App {
	return NewWithClock(fs, time.Now)
}

// NewWithClock returns an App reading time from now instead of the real
// clock — the seam tests use to fix report timestamps.
func NewWithClock(fs fsys.FS, now func() time.Time) App {
	return App{fs: fs, now: now}
}

// nameRE validates track folder names.
var nameRE = regexp.MustCompile(`^[a-z0-9-]+$`)

func unknownStatusErr(status string) error {
	return fmt.Errorf("unknown status %q: want todo, in-progress, done, blocked or backlog", status)
}

// resolveOpenWithRoot resolves id to its open task file and the tree root,
// failing with tree.ErrArchived when id names an archived (read-only) task.
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

// lock takes root's tree-wide write lock, held for a whole use-case so writers
// serialize on the board. Helpers named with a Locked suffix require it held.
func (a App) lock(root string) (func(), error) {
	return a.fs.Lock(filepath.Join(root, names.LockFile))
}

func (a App) lockTree(cwd string) (func(), error) {
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	return a.lock(root)
}

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

func (a App) lockedEdit(root, path string, fn func([]byte) ([]byte, error)) error {
	unlock, err := a.lock(root)
	if err != nil {
		return err
	}
	defer unlock()
	return a.applyEdit(path, fn)
}

// Scope selects the board a board-scoped command targets. In is a root-relative
// track path ("." names the root board); empty means the board containing Cwd —
// the cwd walk-up default.
type Scope struct {
	Cwd string
	In  string
}

func (a App) walkBoards(scope Scope) (boardDir string, boards []string, err error) {
	boardDir, err = a.contextBoard(scope)
	if err != nil {
		return "", nil, err
	}
	boards, err = tree.Boards(a.fs, boardDir)
	if err != nil {
		return "", nil, err
	}
	return boardDir, boards, nil
}

// contextBoard resolves scope to its board directory, validated to exist.
func (a App) contextBoard(scope Scope) (string, error) {
	if scope.In == "" {
		return tree.CurrentBoard(a.fs, scope.Cwd)
	}
	root, err := tree.FindRoot(a.fs, scope.Cwd)
	if err != nil {
		return "", err
	}
	return tree.ResolveTrack(a.fs, root, scope.In)
}

func boardIndexPath(dir string) string {
	return filepath.Join(dir, names.BoardFile)
}

func boardBeside(taskPath string) string {
	return boardIndexPath(filepath.Dir(taskPath))
}

// readTask reads and parses path; read errors pass through unwrapped (callers
// branch on fs.ErrNotExist), parse errors are wrapped with the path.
func (a App) readTask(path string) (task.Task, []byte, error) {
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return task.Task{}, nil, err
	}
	parsed, err := parseTask(path, content)
	if err != nil {
		return task.Task{}, nil, err
	}
	return parsed, content, nil
}

func parseTask(path string, content []byte) (task.Task, error) {
	parsed, err := task.Parse(content)
	if err != nil {
		return task.Task{}, fmt.Errorf("%s: %w", path, err)
	}
	return parsed, nil
}

func (a App) tasksIn(dir string) (files []string, tasks []task.Task, err error) {
	files, err = tree.TaskFileNames(a.fs, dir)
	if err != nil {
		return nil, nil, err
	}
	tasks = make([]task.Task, 0, len(files))
	for _, name := range files {
		parsed, _, err := a.readTask(filepath.Join(dir, name))
		if err != nil {
			return nil, nil, err
		}
		tasks = append(tasks, parsed)
	}
	return files, tasks, nil
}
