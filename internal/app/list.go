package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Row is one open task as read from its file, plus the drift computed
// against its board row.
type Row struct {
	ID         string
	Title      string
	Status     string
	Board      string // slash path of the task's board relative to the listed board; "." when local
	RowMissing bool   // the board index has no row for the task
	RowStatus  string // the board row's status when it disagrees with the file, "" when in sync
}

// List returns one Row per open task in the board containing cwd and every
// board below it, read from the files (never the board index). A non-empty
// status keeps only tasks with that status.
func (a App) List(cwd, status string) ([]Row, error) {
	if status != "" && !task.ValidStatus(status) {
		return nil, unknownStatusErr(status)
	}

	boardDir, err := tree.CurrentBoard(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	boards, err := tree.Boards(a.fs, boardDir)
	if err != nil {
		return nil, err
	}
	var rows []Row
	for _, b := range boards {
		boardRows, err := a.boardRows(boardDir, b)
		if err != nil {
			return nil, err
		}
		for _, r := range boardRows {
			if status != "" && r.Status != status {
				continue
			}
			rows = append(rows, r)
		}
	}
	return rows, nil
}

// boardRows returns one Row per task file directly in board b (its
// sub-boards are separate entries in the caller's board list), tagged with
// its path relative to root — the board list started from.
func (a App) boardRows(root, b string) ([]Row, error) {
	rel, err := filepath.Rel(root, b)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", b, err)
	}
	boardPath := "."
	if rel != "." {
		boardPath = filepath.ToSlash(rel)
	}

	index, err := a.fs.ReadFile(filepath.Join(b, names.BoardFile))
	if err != nil {
		return nil, err
	}
	entries, err := a.fs.ReadDir(b)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", b, err)
	}

	var rows []Row
	for _, e := range entries {
		if e.IsDir() || !tree.IsTaskFile(e.Name()) {
			continue
		}
		path := filepath.Join(b, e.Name())
		content, err := a.fs.ReadFile(path)
		if err != nil {
			return nil, err
		}
		t, err := task.Parse(content)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		row, ok := board.FindRow(index, e.Name())
		missing, rowStatus := drift(ok, row, t.Status)
		rows = append(rows, Row{
			ID: t.ID, Title: t.Title, Status: t.Status,
			Board: boardPath, RowMissing: missing, RowStatus: rowStatus,
		})
	}
	return rows, nil
}

// drift compares a task's file status to its board row, found via
// board.FindRow: rowOK false means the row is missing entirely. A row whose
// status cell disagrees with the file yields that status; in sync (or an
// unreadable cell) yields "".
func drift(rowOK bool, row, fileStatus string) (missing bool, rowStatus string) {
	if !rowOK {
		return true, ""
	}
	s, ok := board.RowStatus(row)
	if !ok || s == fileStatus {
		return false, ""
	}
	return false, s
}
