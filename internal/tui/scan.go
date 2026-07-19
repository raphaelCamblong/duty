// Package tui is duty's live board viewer.
package tui

import (
	"strings"

	"github.com/raphaelCamblong/duty/internal/app"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/task"
)

// Snapshot is the whole tree read at one instant: every board keyed by its
// slash-separated path relative to the root ("." for the root board).
type Snapshot struct {
	Boards map[string]Board
}

// anyInProgress reports whether the whole tree holds at least one in-progress
// task; the root board's rolled-up counts already tally every board below it,
// so this is one map lookup — a snapshot-level answer, not a per-row scan.
func (s Snapshot) anyInProgress() bool {
	return s.Boards["."].Counts[task.StatusInProgress] > 0
}

// Board is one board's view model: identity, its direct tracks, and its
// task rows grouped in board-index section order.
type Board struct {
	Path string
	// Title is the board index H1, falling back to the folder name.
	Title string
	// Parent is the containing board's path, "" for the root.
	Parent string
	// Subs are the direct tracks, in lexical path order.
	Subs []Sub
	// Sections are the task sections in board-index order; task files with no
	// board row are appended to the default section with a drift flag.
	Sections []Section
	// Done and Total count open tasks read from the files — done vs all —
	// in this board and every board below it.
	Done, Total int
	// Counts tallies this subtree's open tasks by status (file truth).
	Counts map[string]int
	// ArchivedCount is the number of archived task files in this board's own
	// archive/ directory — a cheap listing tallied on every scan.
	ArchivedCount int
	// ArchivedSubtree tallies archived tasks in this board and every board
	// below it; link fills it from each board's local ArchivedCount.
	ArchivedSubtree int
	// Archived holds this board's archived task rows, present only when the
	// scan is asked to include them (the TUI's archive toggle); nil otherwise.
	// The dim archived list shows id, title, and age; the normal read-only
	// preview renders each file's body.
	Archived []Row
}

// Sub is one track line of the parent's view, counts live from files.
type Sub struct {
	Path string
	// Name is the folder path relative to the parent, trailing slash.
	Name  string
	Title string
	// Done and Total count the track's subtree like Board's.
	Done, Total int
	// Counts tallies the track's subtree by status, like Board's.
	Counts map[string]int
	// Archived tallies archived tasks across the track's subtree; the OFF
	// hiding rule reads it, the ON view shows it beside the reappeared track.
	Archived int
}

// Section is one "## <name>" group of task rows.
type Section struct {
	Name string
	// Rows are the section's tasks in board order.
	Rows []Row
}

// Row is one task line: the loaded task view read through, with the TUI's own
// drift wording pre-rendered as DriftText — the display reads the words, the
// embedded TaskView.Drift keeps the typed class.
type Row struct {
	app.TaskView
	// DriftText is "" when file and board agree, else "board says <status>",
	// "no row", "no file", or "unparsable file".
	DriftText string
}

// Scan projects the whole tree under root into a Snapshot over one app.Load:
// every board's index and task files joined once, drift and waits computed
// there, archived rows read only when includeArchive is set (the TUI's archive
// toggle), so the default path never reads an archived file's bytes.
func Scan(filesystem fsys.FS, root string, includeArchive bool) (Snapshot, error) {
	view, err := app.New(filesystem).Load(root, app.LoadOptions{Archive: includeArchive})
	if err != nil {
		return Snapshot{}, err
	}
	return project(view), nil
}

// project turns the loaded tree into the display snapshot: one Board per loaded
// board keyed by path, then link rolls subtree counts up and fills each
// parent's Subs. view.Boards is in walk order, which paths inherits.
func project(view *app.TreeView) Snapshot {
	snap := Snapshot{Boards: make(map[string]Board, len(view.Boards))}
	paths := make([]string, 0, len(view.Boards))
	for i := range view.Boards {
		board := projectBoard(view.Boards[i])
		snap.Boards[board.Path] = board
		paths = append(paths, board.Path)
	}
	link(snap, paths)
	return snap
}

func projectBoard(bv app.BoardView) Board {
	board := Board{
		Path:          bv.Path,
		Title:         bv.Title,
		ArchivedCount: bv.ArchivedCount,
		Archived:      projectRows(bv.Archived),
	}
	for i := range bv.Sections {
		board.Sections = append(board.Sections, Section{
			Name: bv.Sections[i].Name,
			Rows: projectRows(bv.Sections[i].Tasks),
		})
	}
	tallyOpen(&board, bv)
	return board
}

func projectRows(tasks []app.TaskView) []Row {
	if len(tasks) == 0 {
		return nil
	}
	rows := make([]Row, len(tasks))
	for i := range tasks {
		rows[i] = Row{TaskView: tasks[i], DriftText: driftText(tasks[i])}
	}
	return rows
}

// driftText is the TUI's one wording for a task's file/board disagreement: ""
// in sync, else the badge text a ⚠ marker prefixes at the render sites.
func driftText(tv app.TaskView) string {
	switch tv.Drift {
	case app.DriftStatus:
		return "board says " + tv.RowStatus
	case app.DriftNoRow:
		return "no row"
	case app.DriftNoFile:
		return "no file"
	case app.DriftBadFile:
		return "unparsable file"
	}
	return ""
}

// tallyOpen fills a board's local Done, Total, and per-status Counts from its
// own open file-truth tasks; link rolls these up into subtree totals.
func tallyOpen(board *Board, bv app.BoardView) {
	board.Counts = make(map[string]int)
	for si := range bv.Sections {
		for ti := range bv.Sections[si].Tasks {
			countOpen(board, bv.Sections[si].Tasks[ti])
		}
	}
}

// countOpen tallies one task into a board's local counts; no-file and bad-file
// rows carry no file truth and never count.
func countOpen(board *Board, tv app.TaskView) {
	if tv.Drift == app.DriftNoFile || tv.Drift == app.DriftBadFile {
		return
	}
	if tv.Status == task.StatusDone {
		board.Done++
	}
	board.Total++
	board.Counts[tv.Status]++
}

// localAgg snapshots one board's own tallies before link rolls each board's
// subtree up from its descendants'.
type localAgg struct {
	done, total, arch int
	counts            map[string]int
}

// link resolves each board's parent, rolls local counts up into subtree
// counts, and fills every parent's Subs. paths is in lexical order, which
// Subs inherits.
func link(snap Snapshot, paths []string) {
	locals := make(map[string]localAgg, len(paths))
	for _, path := range paths {
		current := snap.Boards[path]
		locals[path] = localAgg{done: current.Done, total: current.Total, arch: current.ArchivedCount, counts: current.Counts}
	}
	for _, path := range paths {
		current := rollup(snap.Boards[path], path, paths, locals)
		current.Parent = parentOf(snap, path)
		snap.Boards[path] = current
	}
	buildSubs(snap, paths)
}

// rollup replaces one board's local tallies with the sum over its subtree —
// every board whose path falls within this one.
func rollup(board Board, path string, paths []string, locals map[string]localAgg) Board {
	board.Done, board.Total, board.ArchivedSubtree = 0, 0, 0
	board.Counts = make(map[string]int)
	for _, candidate := range paths {
		if within(candidate, path) {
			accumulate(&board, locals[candidate])
		}
	}
	return board
}

// accumulate adds one descendant board's local tallies into board's subtree totals.
func accumulate(board *Board, lq localAgg) {
	board.Done += lq.done
	board.Total += lq.total
	board.ArchivedSubtree += lq.arch
	for status, count := range lq.counts {
		board.Counts[status] += count
	}
}

func buildSubs(snap Snapshot, paths []string) {
	for _, path := range paths {
		if path == "." {
			continue
		}
		child := snap.Boards[path]
		parent := snap.Boards[child.Parent]
		parent.Subs = append(parent.Subs, Sub{
			Path: path, Name: subName(child.Parent, path), Title: child.Title,
			Done: child.Done, Total: child.Total, Counts: child.Counts,
			Archived: child.ArchivedSubtree,
		})
		snap.Boards[child.Parent] = parent
	}
}

func within(path, prefix string) bool {
	return prefix == "." || path == prefix || strings.HasPrefix(path, prefix+"/")
}

// parentOf returns the nearest ancestor board of p, "." when only the root
// contains it, "" for the root itself.
func parentOf(snap Snapshot, path string) string {
	if path == "." {
		return ""
	}
	segs := strings.Split(path, "/")
	for i := len(segs) - 1; i > 0; i-- {
		cand := strings.Join(segs[:i], "/")
		if _, ok := snap.Boards[cand]; ok {
			return cand
		}
	}
	return "."
}

func subName(parent, path string) string {
	if parent != "." {
		path = strings.TrimPrefix(path, parent+"/")
	}
	return path + "/"
}
