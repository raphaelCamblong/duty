package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
)

// Delete removes an open task: the file, its board row, and any section the
// row's removal leaves empty. A done task is refused unless force is true —
// that's Archive's job.
func (a App) Delete(cwd, id string, force bool) error {
	taskPath, err := a.resolveOpen(cwd, id)
	if err != nil {
		return err
	}
	content, err := a.fs.ReadFile(taskPath)
	if err != nil {
		return err
	}
	t, err := task.Parse(content)
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	if t.Status == task.StatusDone && !force {
		return fmt.Errorf("%s is done: pass --force to delete, or use archive", id)
	}

	boardPath := filepath.Join(filepath.Dir(taskPath), names.BoardFile)
	index, err := a.fs.ReadFile(boardPath)
	if err != nil {
		return err
	}
	dropped, err := board.DropRow(index, filepath.Base(taskPath))
	if err != nil {
		return err
	}
	pruned := board.PruneEmptySections(dropped)

	if err := a.fs.Remove(taskPath); err != nil {
		return fmt.Errorf("delete %s: %w", id, err)
	}
	return a.fs.WriteFile(boardPath, pruned)
}
