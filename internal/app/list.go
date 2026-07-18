package app

import (
	"path/filepath"
	"time"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Row is one open task as read from its file, plus the drift computed
// against its board row.
type Row struct {
	ID         string
	Title      string
	Status     string
	Board      string    // slash path of the task's board relative to the listed board; "." when local
	RowMissing bool      // the board index has no row for the task
	RowStatus  string    // the board row's status when it disagrees with the file, "" when in sync
	UpdatedAt  time.Time // file modification time
	ClaimedBy  string    // agent holding an in-progress task, "" when unclaimed
	Waits      []string  // blocked-by ids not yet done, "" when the task is actionable
}

// List returns one Row per open task in the board in — a root-relative track
// path, or the board containing cwd when empty — and every board below it,
// read from the files (never the board index). A non-empty status keeps only
// tasks with that status.
func (a App) List(cwd, status, in string) ([]Row, error) {
	if status != "" && !task.ValidStatus(status) {
		return nil, unknownStatusErr(status)
	}

	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	boardDir, boards, err := a.walkBoards(cwd, in)
	if err != nil {
		return nil, err
	}
	var rows []Row
	for _, b := range boards {
		boardRows, err := a.boardRows(root, boardDir, b)
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
// tracks are separate entries in the caller's board list), tagged with
// its path relative to listDir — the board list started from. Rows come out
// in board order, mirroring nextInBoard; a file with no board row (drift)
// is appended after, still flagged. treeRoot resolves each row's blocked-by
// ids for its wait state.
func (a App) boardRows(treeRoot, listDir, b string) ([]Row, error) {
	boardPath := relBoard(listDir, b)
	index, err := a.fs.ReadFile(boardIndexPath(b))
	if err != nil {
		return nil, err
	}
	files, err := tree.TaskFileNames(a.fs, b)
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(files))
	for _, name := range boardOrder(index, files) {
		row, err := a.taskRow(treeRoot, index, b, name, boardPath)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// boardOrder sorts files into board order: each present file in the order
// its row appears across index's sections, then any file with no row
// (drift) appended in filename order.
func boardOrder(index []byte, files []string) []string {
	added := make(map[string]bool, len(files))
	for _, name := range files {
		added[name] = false
	}
	ordered := make([]string, 0, len(files))
	for _, sec := range board.Sections(index) {
		for _, r := range sec.Rows {
			if done, ok := added[r.File]; !ok || done {
				continue
			}
			added[r.File] = true
			ordered = append(ordered, r.File)
		}
	}
	for _, name := range files {
		if !added[name] {
			ordered = append(ordered, name)
		}
	}
	return ordered
}

// taskRow assembles filename's Row from its file in dir, its drift computed
// against its row in the board index and its wait state against treeRoot.
func (a App) taskRow(treeRoot string, index []byte, dir, filename, boardPath string) (Row, error) {
	path := filepath.Join(dir, filename)
	t, _, err := a.readTask(path)
	if err != nil {
		return Row{}, err
	}
	waits, err := a.unmetDeps(treeRoot, t.BlockedBy)
	if err != nil {
		return Row{}, err
	}
	row, ok := board.FindRow(index, filename)
	missing, rowStatus := drift(ok, row, t.Status)
	return Row{
		ID: t.ID, Title: t.Title, Status: t.Status,
		Board: boardPath, RowMissing: missing, RowStatus: rowStatus,
		Waits:     waits,
		UpdatedAt: a.mtime(path), ClaimedBy: t.ClaimedBy,
	}, nil
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
