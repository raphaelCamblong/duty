package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
)

// SetStatus sets a task's status: the frontmatter `status:` line and the
// board row's status cell change in one use-case (the sync invariant).
// Unknown statuses and archived ids are rejected; both new contents are
// computed before either file is written.
func (a App) SetStatus(cwd, id, status string) error {
	if !task.ValidStatus(status) {
		return unknownStatusErr(status)
	}

	taskPath, err := a.resolveOpen(cwd, id)
	if err != nil {
		return err
	}
	content, err := a.fs.ReadFile(taskPath)
	if err != nil {
		return err
	}
	updated, err := task.SetStatus(content, status)
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	boardPath := filepath.Join(filepath.Dir(taskPath), names.BoardFile)
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
