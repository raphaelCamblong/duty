package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Archive moves every done task in scope and below into its board's archive/
// (row dropped, footer rewritten); a board with nothing to archive is untouched.
func (a App) Archive(s Scope) error {
	unlock, err := a.lockTree(s.Cwd)
	if err != nil {
		return err
	}
	defer unlock()
	_, boards, err := a.walkBoards(s)
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

func (a App) archiveBoard(b string) error {
	done, err := a.doneTasks(b)
	if err != nil {
		return err
	}
	if len(done) == 0 {
		return nil
	}
	boardPath := boardIndexPath(b)
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

func dropRows(index []byte, filenames []string) ([]byte, error) {
	for _, name := range filenames {
		var err error
		if index, err = board.DropRow(index, name); err != nil {
			return nil, err
		}
	}
	return board.PruneEmptySections(index), nil
}

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

func (a App) doneTasks(dir string) ([]string, error) {
	files, tasks, err := a.tasksIn(dir)
	if err != nil {
		return nil, err
	}
	var done []string
	for i, t := range tasks {
		if t.Status == task.StatusDone {
			done = append(done, files[i])
		}
	}
	return done, nil
}

func (a App) countTaskFiles(dir string) (int, error) {
	files, err := tree.TaskFileNames(a.fs, dir)
	if err != nil {
		return 0, err
	}
	return len(files), nil
}
