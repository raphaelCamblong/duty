package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/task"
)

// StatusChange is a request to set a task's status: its id, the target status,
// the As name to record when the status is in-progress, and Force to take over
// an existing in-progress claim.
type StatusChange struct {
	ID     string
	Status string
	As     string
	Force  bool
}

// SetStatus flips a task's status in both the file and its board row (the sync
// invariant) under the tree lock; unknown statuses and archived ids are rejected.
func (a App) SetStatus(cwd string, ch StatusChange) error {
	if !task.ValidStatus(ch.Status) {
		return unknownStatusErr(ch.Status)
	}
	root, taskPath, err := a.resolveOpenWithRoot(cwd, ch.ID)
	if err != nil {
		return err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return err
	}
	defer unlock()
	return a.setStatusLocked(taskPath, ch)
}

// setStatusLocked computes both new file contents before writing either.
func (a App) setStatusLocked(taskPath string, ch StatusChange) error {
	parsed, content, err := a.readTask(taskPath)
	if err != nil {
		return err
	}
	return a.statusWrite(taskPath, ch, content, parsed)
}

// statusWrite applies ch onto already-read content (possibly pre-edited by
// report) and cur, computing both file writes before either — an error leaves both.
func (a App) statusWrite(taskPath string, ch StatusChange, content []byte, cur task.Task) error {
	if err := guardClaim(ch, cur.Status, cur.ClaimedBy); err != nil {
		return err
	}
	updated, err := task.SetStatus(content, ch.Status)
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	updated, err = task.SetClaimedBy(updated, claimerFor(ch.Status, ch.As))
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	boardPath := boardBeside(taskPath)
	index, err := a.fs.ReadFile(boardPath)
	if err != nil {
		return err
	}
	withCell, err := board.SetRowStatus(index, filepath.Base(taskPath), ch.Status)
	if err != nil {
		return err
	}
	if err := a.fs.WriteFile(taskPath, updated); err != nil {
		return err
	}
	return a.fs.WriteFile(boardPath, withCell)
}

// claimerFor returns the recorded claimer: as when moving to in-progress, empty
// otherwise — the field only ever means "currently holds it".
func claimerFor(status, as string) string {
	if status == task.StatusInProgress {
		return as
	}
	return ""
}

// guardClaim refuses to re-claim a task already in-progress (current) without
// Force, naming holder when one is recorded; every other transition is free.
func guardClaim(ch StatusChange, current, holder string) error {
	if ch.Force || ch.Status != task.StatusInProgress || current != task.StatusInProgress {
		return nil
	}
	if holder != "" {
		return fmt.Errorf("%s is claimed by %s — use --force to take it over", ch.ID, holder)
	}
	return fmt.Errorf("%s is already in-progress — someone claimed it; use --force to take it over", ch.ID)
}
