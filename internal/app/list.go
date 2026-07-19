package app

import (
	"time"

	"github.com/raphaelCamblong/duty/internal/task"
)

// Row is one task as the board lists it: file truth joined with the drift
// computed against its board row.
type Row struct {
	ID        string
	Title     string
	Status    string
	Board     string    // slash path of the task's board relative to the listed base; "." when local
	Drift     Drift     // typed file/board disagreement, DriftNone when in sync
	RowStatus string    // the board row's status when it disagrees or has no file, "" when in sync
	UpdatedAt time.Time // file modification time
	ClaimedBy string    // agent holding an in-progress task, "" when unclaimed
	Waits     []string  // blocked-by ids not yet met, empty when the task is actionable
}

// List returns one Row per task in scope's board and below, in board order:
// each board's sections flattened in index order, strays already sorted into
// the default section. A non-empty status keeps only that status.
func (a App) List(scope Scope, status string) ([]Row, error) {
	if status != "" && !task.ValidStatus(status) {
		return nil, unknownStatusErr(status)
	}
	view, err := a.Load(scope.Cwd, LoadOptions{})
	if err != nil {
		return nil, err
	}
	base, err := view.ScopeBase(scope)
	if err != nil {
		return nil, err
	}
	boards := view.Under(base)
	baseDir := boards[0].Dir
	var rows []Row
	for i := range boards {
		rows = append(rows, listBoard(baseDir, boards[i], status)...)
	}
	return rows, nil
}

func listBoard(baseDir string, bv BoardView, status string) []Row {
	boardPath := relBoard(baseDir, bv.Dir)
	var rows []Row
	for _, tv := range bv.tasks() {
		if status == "" || tv.Status == status {
			rows = append(rows, rowOf(tv, boardPath))
		}
	}
	return rows
}

func rowOf(tv TaskView, boardPath string) Row {
	return Row{
		ID: tv.ID, Title: tv.Title, Status: tv.Status, Board: boardPath,
		Drift: tv.Drift, RowStatus: tv.RowStatus,
		UpdatedAt: tv.UpdatedAt, ClaimedBy: tv.ClaimedBy, Waits: tv.Waits,
	}
}
