package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Archive moves every status: done task in the board in — a root-relative
// track path, or the board containing cwd when empty — and every board below
// it into its own board's archive/: the file is renamed, its row dropped,
// empty sections pruned, and that board's footer count rewritten. A board
// with nothing to archive is left untouched, which makes a second run a clean
// no-op.
func (a App) Archive(cwd, in string) error {
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return err
	}
	defer unlock()
	_, boards, err := a.walkBoards(cwd, in)
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
	index, err = dropRows(index, done)
	if err != nil {
		return err
	}
	count, err := a.moveToArchive(b, done)
	if err != nil {
		return err
	}
	index, err = board.SetArchivedCount(index, count)
	if err != nil {
		return err
	}
	return a.fs.WriteFile(boardPath, index)
}

// dropRows drops one row per filename from the board index and prunes any
// section left empty.
func dropRows(index []byte, filenames []string) ([]byte, error) {
	for _, name := range filenames {
		var err error
		if index, err = board.DropRow(index, name); err != nil {
			return nil, err
		}
	}
	return board.PruneEmptySections(index), nil
}

// moveToArchive renames every named task file in b into b's archive/,
// creating it if needed, and returns the archive's new task-file count.
func (a App) moveToArchive(b string, filenames []string) (int, error) {
	archiveDir := filepath.Join(b, names.ArchiveDir)
	if err := a.fs.MkdirAll(archiveDir); err != nil {
		return 0, fmt.Errorf("archive %s: %w", b, err)
	}
	for _, name := range filenames {
		if err := a.fs.Rename(filepath.Join(b, name), filepath.Join(archiveDir, name)); err != nil {
			return 0, fmt.Errorf("archive %s: %w", name, err)
		}
	}
	return a.countTaskFiles(archiveDir)
}

// doneTasks returns the filenames of every status: done task filed directly
// in dir.
func (a App) doneTasks(dir string) ([]string, error) {
	files, err := tree.TaskFileNames(a.fs, dir)
	if err != nil {
		return nil, err
	}
	var done []string
	for _, name := range files {
		t, _, err := a.readTask(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if t.Status == task.StatusDone {
			done = append(done, name)
		}
	}
	return done, nil
}

// countTaskFiles counts the task files directly in dir.
func (a App) countTaskFiles(dir string) (int, error) {
	files, err := tree.TaskFileNames(a.fs, dir)
	if err != nil {
		return 0, err
	}
	return len(files), nil
}
