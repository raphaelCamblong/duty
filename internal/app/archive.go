package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Archive moves every status: done task in the board containing cwd and
// every board below it into its own board's archive/: the file is renamed,
// its row dropped, empty sections pruned, and that board's footer count
// rewritten. A board with nothing to archive is left untouched, which makes
// a second run a clean no-op.
func (a App) Archive(cwd string) error {
	boardDir, err := tree.CurrentBoard(a.fs, cwd)
	if err != nil {
		return err
	}
	boards, err := tree.Boards(a.fs, boardDir)
	if err != nil {
		return err
	}
	for _, b := range boards {
		if err := a.archiveBoard(b); err != nil {
			return err
		}
	}
	return nil
}

// archiveBoard archives every done task filed directly in b (not in a
// track) into b's own archive/ directory.
func (a App) archiveBoard(b string) error {
	done, err := a.doneTasks(b)
	if err != nil {
		return err
	}
	if len(done) == 0 {
		return nil
	}

	boardPath := filepath.Join(b, names.BoardFile)
	index, err := a.fs.ReadFile(boardPath)
	if err != nil {
		return err
	}
	for _, name := range done {
		if index, err = board.DropRow(index, name); err != nil {
			return err
		}
	}
	index = board.PruneEmptySections(index)

	archiveDir := filepath.Join(b, names.ArchiveDir)
	if err := a.fs.MkdirAll(archiveDir); err != nil {
		return fmt.Errorf("archive %s: %w", b, err)
	}
	for _, name := range done {
		if err := a.fs.Rename(filepath.Join(b, name), filepath.Join(archiveDir, name)); err != nil {
			return fmt.Errorf("archive %s: %w", name, err)
		}
	}
	count, err := a.countTaskFiles(archiveDir)
	if err != nil {
		return err
	}
	index, err = board.SetArchivedCount(index, count)
	if err != nil {
		return err
	}
	return a.fs.WriteFile(boardPath, index)
}

// doneTasks returns the filenames of every status: done task filed directly
// in dir.
func (a App) doneTasks(dir string) ([]string, error) {
	entries, err := a.fs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("archive %s: %w", dir, err)
	}
	var done []string
	for _, e := range entries {
		if e.IsDir() || !tree.IsTaskFile(e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		content, err := a.fs.ReadFile(path)
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
func (a App) countTaskFiles(dir string) (int, error) {
	entries, err := a.fs.ReadDir(dir)
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
