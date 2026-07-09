package app

import (
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
)

// Link moves a task's board row under "## <section>", creating the section
// above the footer when absent and pruning any section left empty. Sections
// live only on the board — the task file carries none — so this is the one
// mutation that touches a single file.
func (a App) Link(cwd, id, section string) error {
	taskPath, err := a.resolveOpen(cwd, id)
	if err != nil {
		return err
	}
	boardPath := filepath.Join(filepath.Dir(taskPath), names.BoardFile)
	index, err := a.fs.ReadFile(boardPath)
	if err != nil {
		return err
	}
	moved, err := board.MoveRow(index, filepath.Base(taskPath), section)
	if err != nil {
		return err
	}
	return a.fs.WriteFile(boardPath, board.PruneEmptySections(moved))
}
