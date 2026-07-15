// Package app implements duty's use-cases: one method per verb, all
// orchestration over an injected fsys.FS. The sync invariant lives here —
// every mutating use-case edits the task file AND its board row in one call.
// Methods return data and errors; they never print and never parse flags.
package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// App bundles duty's use-cases over one filesystem.
type App struct {
	fs fsys.FS
}

// New returns an App operating on f.
func New(f fsys.FS) App {
	return App{fs: f}
}

// nameRE validates track folder names and task filename slugs.
var nameRE = regexp.MustCompile(`^[a-z0-9-]+$`)

// unknownStatusErr is the one-line error every use-case rejecting an unknown
// status string returns.
func unknownStatusErr(status string) error {
	return fmt.Errorf("unknown status %q: want todo, in-progress, done or blocked", status)
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

// resolveOpen resolves id to its open task file anywhere in the tree
// containing cwd, discarding the tree root.
func (a App) resolveOpen(cwd, id string) (string, error) {
	_, path, err := a.resolveOpenWithRoot(cwd, id)
	return path, err
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
	return a.resolveTrack(root, in)
}

// resolveTrack resolves track — a slash path relative to root, "." naming the
// root board — to an existing board directory inside the tree. It is the one
// track-path validator, shared by move --track and contextBoard: an absolute
// or escaping path and a path naming no board are the one same failure,
// unknown track %q.
func (a App) resolveTrack(root, track string) (string, error) {
	dir := filepath.Join(root, filepath.FromSlash(track))
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", unknownTrackErr(track)
	}
	info, err := a.fs.Stat(filepath.Join(dir, names.BoardFile))
	if err != nil || info.IsDir() {
		return "", unknownTrackErr(track)
	}
	return dir, nil
}

// unknownTrackErr is the one-line error a root-relative track path naming no
// board in the tree returns, shared by move --track and contextBoard.
func unknownTrackErr(track string) error {
	return fmt.Errorf("unknown track %q", track)
}

// boardBeside returns the path of the board index in the same directory as
// the task file at taskPath.
func boardBeside(taskPath string) string {
	return filepath.Join(filepath.Dir(taskPath), names.BoardFile)
}

// readTask reads and parses the task file at path, returning the parsed task
// and the raw contents. Read errors pass through unwrapped so callers can
// branch on fs.ErrNotExist; parse errors carry the path.
func (a App) readTask(path string) (task.Task, []byte, error) {
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return task.Task{}, nil, err
	}
	t, err := task.Parse(content)
	if err != nil {
		return task.Task{}, nil, fmt.Errorf("%s: %w", path, err)
	}
	return t, content, nil
}
