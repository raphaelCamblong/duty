package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/task"
)

// Delete removes an open task: the file, its board row, and any section the
// row's removal leaves empty. A done task is refused unless force is true —
// that's Archive's job.
func (a App) Delete(cwd, id string, force bool) error {
	root, taskPath, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return err
	}
	defer unlock()
	return a.deleteLocked(id, taskPath, force)
}

// deleteLocked removes an open task's file and board row, refusing a done task
// unless force is set. It must run under the tree lock.
func (a App) deleteLocked(id, taskPath string, force bool) error {
	t, _, err := a.readTask(taskPath)
	if err != nil {
		return err
	}
	if t.Status == task.StatusDone && !force {
		return fmt.Errorf("%s is done: pass --force to delete, or use archive", id)
	}
	boardPath := boardBeside(taskPath)
	pruned, err := a.dropFromBoard(boardPath, filepath.Base(taskPath))
	if err != nil {
		return err
	}
	if err := a.fs.Remove(taskPath); err != nil {
		return fmt.Errorf("delete %s: %w", id, err)
	}
	return a.fs.WriteFile(boardPath, pruned)
}
