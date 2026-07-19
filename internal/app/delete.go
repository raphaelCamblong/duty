package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/task"
)

// Delete removes an open task — its file, board row, and any emptied section;
// a done task is refused unless force is true (that's Archive's job).
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

func (a App) deleteLocked(id, taskPath string, force bool) error {
	parsed, _, err := a.readTask(taskPath)
	if err != nil {
		return err
	}
	if parsed.Status == task.StatusDone && !force {
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
