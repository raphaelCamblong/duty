package cli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
)

const deleteUsage = "usage: duty delete <id> [--force]"

// runDelete removes an open task: the file, its board row, and any section
// the row's removal leaves empty. A done task is refused unless --force is
// given — that's archive's job.
func runDelete(f fsys.FS, cwd string, args []string) error {
	set := flag.NewFlagSet("delete", flag.ContinueOnError)
	force := set.Bool("force", false, "allow deleting a done task")
	pos, err := positionals(set, args, deleteUsage)
	if err != nil {
		return err
	}
	if len(pos) != 1 || pos[0] == "" {
		return errors.New(deleteUsage)
	}
	id := pos[0]

	taskPath, err := resolveOpen(f, cwd, id)
	if err != nil {
		return err
	}
	content, err := f.ReadFile(taskPath)
	if err != nil {
		return err
	}
	t, err := task.Parse(content)
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	if t.Status == task.StatusDone && !*force {
		return fmt.Errorf("%s is done: pass --force to delete, or use archive", id)
	}

	boardPath := filepath.Join(filepath.Dir(taskPath), names.BoardFile)
	index, err := f.ReadFile(boardPath)
	if err != nil {
		return err
	}
	dropped, err := board.DropRow(index, filepath.Base(taskPath))
	if err != nil {
		return err
	}
	pruned := board.PruneEmptySections(dropped)

	if err := f.Remove(taskPath); err != nil {
		return fmt.Errorf("delete %s: %w", id, err)
	}
	return f.WriteFile(boardPath, pruned)
}
