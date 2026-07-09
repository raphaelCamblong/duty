package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/fsutil"
	"github.com/raphaelCamblong/duty/internal/task"
)

const statusUsage = "usage: duty status <id> <status>"

// runStatus sets a task's status: the frontmatter `status:` line and the
// board row's status cell change in one command (the sync invariant).
// Unknown statuses and archived ids are rejected; both new contents are
// computed before either file is written.
func runStatus(cwd string, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	pos, err := positionals(fs, args, statusUsage)
	if err != nil {
		return err
	}
	if len(pos) != 2 || pos[0] == "" || pos[1] == "" {
		return errors.New(statusUsage)
	}
	id, status := pos[0], pos[1]
	if !task.ValidStatus(status) {
		return unknownStatusErr(status)
	}

	taskPath, err := resolveOpen(cwd, id)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(taskPath)
	if err != nil {
		return err
	}
	updated, err := task.SetStatus(content, status)
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	boardPath := filepath.Join(filepath.Dir(taskPath), boardFile)
	index, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}
	withCell, err := board.SetRowStatus(index, filepath.Base(taskPath), status)
	if err != nil {
		return err
	}

	if err := fsutil.WriteAtomic(taskPath, updated); err != nil {
		return err
	}
	return fsutil.WriteAtomic(boardPath, withCell)
}
