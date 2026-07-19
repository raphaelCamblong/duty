package app

import (
	"fmt"
	"maps"
	"slices"
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

// Snapshot reads every open task in scope and below into the state duty watch
// diffs, keyed by id. No-file and bad-file rows carry no file truth, so watch
// skips them — a half-saved file emits no event until it parses.
func (a App) Snapshot(scope Scope) (map[string]TaskState, error) {
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
	states := make(map[string]TaskState)
	for i := range boards {
		snapshotBoard(states, baseDir, boards[i])
	}
	return states, nil
}

func snapshotBoard(states map[string]TaskState, baseDir string, bv BoardView) {
	boardPath := relBoard(baseDir, bv.Dir)
	for _, tv := range bv.tasks() {
		if tv.hasFileTruth() {
			states[tv.ID] = TaskState{
				Status: tv.Status, ClaimedBy: tv.ClaimedBy, Board: boardPath,
				GatesDone: tv.GatesDone, GatesTotal: tv.GatesTotal,
			}
		}
	}
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

func gatePair(state TaskState) string {
	return fmt.Sprintf("%d/%d", state.GatesDone, state.GatesTotal)
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
