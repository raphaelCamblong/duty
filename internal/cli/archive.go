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
	"github.com/raphaelCamblong/duty/internal/tree"
)

const archiveUsage = "usage: duty archive"

// runArchive moves every status: done task in the current board and every
// board below it into its own board's archive/: the file is renamed, its row
// dropped, empty sections pruned, and that board's footer count rewritten. A
// board with nothing to archive is left untouched, which makes a second run a
// clean no-op.
func runArchive(cwd string, args []string) error {
	fs := flag.NewFlagSet("archive", flag.ContinueOnError)
	pos, err := positionals(fs, args, archiveUsage)
	if err != nil {
		return err
	}
	if len(pos) != 0 {
		return errors.New(archiveUsage)
	}

	boardDir, err := tree.CurrentBoard(cwd)
	if err != nil {
		return err
	}
	boards, err := tree.Boards(boardDir)
	if err != nil {
		return err
	}
	for _, b := range boards {
		if err := archiveBoard(b); err != nil {
			return err
		}
	}
	return nil
}

// archiveBoard archives every done task filed directly in b (not in a
// sub-board) into b's own archive/ directory.
func archiveBoard(b string) error {
	done, err := doneTasks(b)
	if err != nil {
		return err
	}
	if len(done) == 0 {
		return nil
	}

	boardPath := filepath.Join(b, boardFile)
	index, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}
	for _, name := range done {
		if index, err = board.DropRow(index, name); err != nil {
			return err
		}
	}
	index = board.PruneEmptySections(index)

	archiveDir := filepath.Join(b, tree.ArchiveDir)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return fmt.Errorf("archive %s: %w", b, err)
	}
	for _, name := range done {
		if err := os.Rename(filepath.Join(b, name), filepath.Join(archiveDir, name)); err != nil {
			return fmt.Errorf("archive %s: %w", name, err)
		}
	}
	count, err := countTaskFiles(archiveDir)
	if err != nil {
		return err
	}
	index, err = board.SetArchivedCount(index, count)
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(boardPath, index)
}

// doneTasks returns the filenames of every status: done task filed directly
// in dir.
func doneTasks(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("archive %s: %w", dir, err)
	}
	var done []string
	for _, e := range entries {
		if e.IsDir() || !tree.IsTaskFile(e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		t, err := task.Parse(content)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if t.Status == task.StatusDone {
			done = append(done, e.Name())
		}
	}
	return done, nil
}

// countTaskFiles counts the task files directly in dir.
func countTaskFiles(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("count %s: %w", dir, err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && tree.IsTaskFile(e.Name()) {
			n++
		}
	}
	return n, nil
}
