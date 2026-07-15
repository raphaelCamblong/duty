package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/task"
)

// SetStatus sets a task's status: the frontmatter `status:` line and the
// board row's status cell change in one use-case (the sync invariant), under
// the tree write lock. Unknown statuses and archived ids are rejected;
// re-claiming an already in-progress task is refused unless force is set.
func (a App) SetStatus(cwd, id, status string, force bool) error {
	if !task.ValidStatus(status) {
		return unknownStatusErr(status)
	}
	root, taskPath, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return err
	}
	defer unlock()
	return a.setStatusLocked(taskPath, id, status, force)
}

// setStatusLocked performs the synced status write; both new contents are
// computed before either file is written. It must run under the tree lock.
func (a App) setStatusLocked(taskPath, id, status string, force bool) error {
	t, content, err := a.readTask(taskPath)
	if err != nil {
		return err
	}
	if err := guardClaim(id, status, t.Status, force); err != nil {
		return err
	}
	updated, err := task.SetStatus(content, status)
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	boardPath := boardBeside(taskPath)
	index, err := a.fs.ReadFile(boardPath)
	if err != nil {
		return err
	}
	withCell, err := board.SetRowStatus(index, filepath.Base(taskPath), status)
	if err != nil {
		return err
	}
	if err := a.fs.WriteFile(taskPath, updated); err != nil {
		return err
	}
	return a.fs.WriteFile(boardPath, withCell)
}

// guardClaim refuses to take over a task already in-progress when a caller
// sets it in-progress again without force; every other transition is free
// (the flat setter of spec §3).
func guardClaim(id, status, current string, force bool) error {
	if !force && status == task.StatusInProgress && current == task.StatusInProgress {
		return fmt.Errorf("%s is already in-progress — someone claimed it; use --force to take it over", id)
	}
	return nil
}
