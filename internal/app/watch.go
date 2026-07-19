package app

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// TaskState is one open task's watched fields at a moment — the values duty
// watch diffs between consecutive scans.
type TaskState struct {
	Status     string
	ClaimedBy  string
	Board      string
	GatesDone  int
	GatesTotal int
}

// Event kinds: an Event's Kind and the second column of its --agent record.
const (
	EventCreated   = "created"
	EventDeleted   = "deleted"
	EventStatus    = "status"
	EventClaimedBy = "claimed-by"
	EventMoved     = "moved"
	EventGates     = "gates"
)

type Event struct {
	Kind  string
	ID    string
	Field string
	Old   string
	New   string
}

// Snapshot reads every open task in scope and below, keyed by id — the state
// duty watch diffs; unparsable files are skipped.
func (a App) Snapshot(s Scope) (map[string]TaskState, error) {
	boardDir, boards, err := a.walkBoards(s)
	if err != nil {
		return nil, err
	}
	states := make(map[string]TaskState)
	for _, b := range boards {
		if err := a.boardStates(boardDir, b, states); err != nil {
			return nil, err
		}
	}
	return states, nil
}

// boardStates reads every task file directly in b into states, keyed by id and
// tagged with b's path relative to listDir — the board the snapshot started from.
func (a App) boardStates(listDir, b string, states map[string]TaskState) error {
	boardPath := relBoard(listDir, b)
	files, err := tree.TaskFileNames(a.fs, b)
	if err != nil {
		return err
	}
	for _, name := range files {
		content, err := a.fs.ReadFile(filepath.Join(b, name))
		if err != nil {
			return err
		}
		t, err := task.Parse(content)
		if err != nil {
			continue
		}
		gd, gt := task.CountGates(content)
		states[t.ID] = TaskState{
			Status: t.Status, ClaimedBy: t.ClaimedBy, Board: boardPath,
			GatesDone: gd, GatesTotal: gt,
		}
	}
	return nil
}

// Diff returns the changes from before to after — one Event per changed field —
// ordered by task id then a fixed field order, so the stream is deterministic.
func Diff(before, after map[string]TaskState) []Event {
	var events []Event
	for _, id := range sortedIDs(before, after) {
		prev, had := before[id]
		cur, has := after[id]
		switch {
		case !had:
			events = append(events, Event{Kind: EventCreated, ID: id, Field: "status", New: cur.Status})
		case !has:
			events = append(events, Event{Kind: EventDeleted, ID: id, Field: "status", Old: prev.Status})
		default:
			events = append(events, changedFields(id, prev, cur)...)
		}
	}
	return events
}

// changedFields lists one Event per field that differs between prev and cur,
// in the fixed order status, claimed-by, board, gates.
func changedFields(id string, prev, cur TaskState) []Event {
	var events []Event
	if prev.Status != cur.Status {
		events = append(events, Event{Kind: EventStatus, ID: id, Field: "status", Old: prev.Status, New: cur.Status})
	}
	if prev.ClaimedBy != cur.ClaimedBy {
		events = append(events, Event{Kind: EventClaimedBy, ID: id, Field: "claimed-by", Old: prev.ClaimedBy, New: cur.ClaimedBy})
	}
	if prev.Board != cur.Board {
		events = append(events, Event{Kind: EventMoved, ID: id, Field: "board", Old: prev.Board, New: cur.Board})
	}
	if prev.GatesDone != cur.GatesDone || prev.GatesTotal != cur.GatesTotal {
		events = append(events, Event{Kind: EventGates, ID: id, Field: "gates", Old: gatePair(prev), New: gatePair(cur)})
	}
	return events
}

func gatePair(s TaskState) string {
	return fmt.Sprintf("%d/%d", s.GatesDone, s.GatesTotal)
}

func sortedIDs(before, after map[string]TaskState) []string {
	ids := make(map[string]struct{}, len(before)+len(after))
	for id := range before {
		ids[id] = struct{}{}
	}
	for id := range after {
		ids[id] = struct{}{}
	}
	return slices.Sorted(maps.Keys(ids))
}
