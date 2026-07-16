package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/task"
)

// SetStatus sets a task's status: the frontmatter `status:` line and the
// board row's status cell change in one use-case (the sync invariant), under
// the tree write lock. Moving to in-progress records as as the claimer (empty
// leaves the claim unnamed); any other status clears the claim. Unknown
// statuses and archived ids are rejected; re-claiming an already in-progress
// task is refused unless force is set.
func (a App) SetStatus(cwd, id, status string, force bool, as string) error {
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
	return a.setStatusLocked(taskPath, id, status, force, as)
}

// setStatusLocked performs the synced status write; both new contents are
// computed before either file is written. It must run under the tree lock.
func (a App) setStatusLocked(taskPath, id, status string, force bool, as string) error {
	t, content, err := a.readTask(taskPath)
	if err != nil {
		return err
	}
	return a.statusWrite(taskPath, id, status, force, content, t.Status, t.ClaimedBy, as)
}

// statusWrite applies the synced status change onto content — the task file's
// bytes, already read (and possibly already edited, as by report --status) —
// with current its parsed status and holder its current claimer: it guards the
// claim, rewrites the status line, sets or clears the claimed-by line, rewrites
// the board cell, then writes both files, every new content computed before
// either write so an error leaves both untouched. It must run under the tree lock.
func (a App) statusWrite(taskPath, id, status string, force bool, content []byte, current, holder, as string) error {
	if err := guardClaim(id, status, current, holder, force); err != nil {
		return err
	}
	updated, err := task.SetStatus(content, status)
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	updated, err = task.SetClaimedBy(updated, claimerFor(status, as))
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

// claimerFor is the claimed-by name a status write records: as when the task is
// moving to in-progress, empty otherwise — the field only ever means "currently
// holds it", so every other status clears it.
func claimerFor(status, as string) string {
	if status == task.StatusInProgress {
		return as
	}
	return ""
}

// guardClaim refuses to take over a task already in-progress when a caller
// sets it in-progress again without force; every other transition is free
// (the flat setter of spec §3). The refusal names the current holder when one
// is recorded.
func guardClaim(id, status, current, holder string, force bool) error {
	if force || status != task.StatusInProgress || current != task.StatusInProgress {
		return nil
	}
	if holder != "" {
		return fmt.Errorf("%s is claimed by %s — use --force to take it over", id, holder)
	}
	return fmt.Errorf("%s is already in-progress — someone claimed it; use --force to take it over", id)
}
