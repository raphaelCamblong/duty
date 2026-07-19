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

// SetStatus sets a task's status: the frontmatter `status:` line and the
// board row's status cell change in one use-case (the sync invariant), under
// the tree write lock. Moving to in-progress records As as the claimer (empty
// leaves the claim unnamed); any other status clears the claim. Unknown
// statuses and archived ids are rejected; re-claiming an already in-progress
// task is refused unless Force is set.
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

// setStatusLocked computes both new file contents before writing either. It
// must run under the tree lock.
func (a App) setStatusLocked(taskPath string, ch StatusChange) error {
	t, content, err := a.readTask(taskPath)
	if err != nil {
		return err
	}
	return a.statusWrite(taskPath, ch, content, t)
}

// statusWrite applies ch onto content — the task file's bytes, already read
// (and possibly already edited, as by report --status) — with cur the task's
// current parsed state (its status and claimer). Every new content is computed
// before either write, so an error leaves both untouched. It must run under the
// tree lock.
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

// claimerFor is the claimed-by name a status write records: as when the task is
// moving to in-progress, empty otherwise — the field only ever means "currently
// holds it", so every other status clears it.
func claimerFor(status, as string) string {
	if status == task.StatusInProgress {
		return as
	}
	return ""
}

// guardClaim refuses to take over a task already in-progress when ch sets it
// in-progress again without Force; every other transition is free (the flat
// setter of spec §3). current and holder are the task's present status and
// claimer; the refusal names the holder when one is recorded.
func guardClaim(ch StatusChange, current, holder string) error {
	if ch.Force || ch.Status != task.StatusInProgress || current != task.StatusInProgress {
		return nil
	}
	if holder != "" {
		return fmt.Errorf("%s is claimed by %s — use --force to take it over", ch.ID, holder)
	}
	return fmt.Errorf("%s is already in-progress — someone claimed it; use --force to take it over", ch.ID)
}
