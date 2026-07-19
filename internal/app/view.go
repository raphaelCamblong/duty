package app

import (
	"fmt"
	"strings"
	"time"
)

// TreeView is the whole duty tree loaded once: every board joined to its task
// files, drift and waits computed, ready for list/get/next/watch/tui to project
// over. Load builds it; nothing here touches the filesystem.
type TreeView struct {
	// Root is the tree root's absolute path.
	Root string
	// Boards are every board in the tree in tree.Boards walk order (lexical),
	// the root board first.
	Boards []BoardView

	byID        map[string]*TaskView
	archivedIDs map[string]bool
	// contextPath is the board containing the cwd Load ran from — the empty
	// scope's base.
	contextPath string
}

// TaskView is one task as the system sees it: file truth, board truth, and the
// drift/waits duty computes between them, joined once.
type TaskView struct {
	// File truth — the task file. Path is "" and these fields are zero when a
	// board row points at no file.
	ID         string
	Title      string
	Status     string
	BlockedBy  []string
	ClaimedBy  string
	GatesDone  int
	GatesTotal int
	Content    []byte
	Path       string
	UpdatedAt  time.Time

	// Board truth — the row indexing this task. File is the row's target
	// filename; RowStatus is the row's status cell when it disagrees with the
	// file or when there is no file, "" when file and row agree.
	File      string
	RowStatus string

	// Computed. Drift names any file/board disagreement; Waits are the unmet
	// blocked-by ids; Deps pairs every blocked-by id with its computed status.
	Drift Drift
	Waits []string
	Deps  []Dep
}

// Drift classifies the disagreement between a task's file and its board row.
// Formatters own the words shown for each class; the model only names it.
type Drift int

const (
	// DriftNone is a file and row that agree, or an archived task.
	DriftNone Drift = iota
	// DriftNoRow is a task file with no board row.
	DriftNoRow
	// DriftStatus is a board row whose status cell disagrees with the file.
	DriftStatus
	// DriftNoFile is a board row pointing at a file that does not exist.
	DriftNoFile
	// DriftBadFile is a file whose frontmatter will not parse.
	DriftBadFile
)

// BoardView is one board: its identity, its task sections in index order, and
// its archive tally.
type BoardView struct {
	// Dir is the board's absolute directory.
	Dir string
	// Path is Dir relative to the root in slash form, "." for the root board.
	Path string
	// Title is the index H1, the folder name when it has none.
	Title string
	// Sections are the task sections in board-index order; rowless files sort
	// by name into the default section, appended last when the index has none.
	Sections []SectionView
	// ArchivedCount is the number of archived task files in this board's
	// archive/ directory, always tallied.
	ArchivedCount int
	// Archived holds this board's archived tasks, read only when Load is asked
	// to include them; empty otherwise.
	Archived []TaskView
}

// SectionView is one "## <name>" group of tasks in board order.
type SectionView struct {
	Name  string
	Tasks []TaskView
}

// Task returns the open task with the given id, resolved from memory. Only
// file-truth tasks answer: a no-file or unparsable drift row does not.
func (view *TreeView) Task(id string) (*TaskView, bool) {
	found, ok := view.byID[id]
	return found, ok
}

// ScopeBase resolves scope to the board its command walks from: "." the root,
// "" the board Load ran in, else a named track. A named track no loaded board
// matches is the one error, mirroring an unknown --in.
func (view *TreeView) ScopeBase(scope Scope) (string, error) {
	if scope.In == "." {
		return ".", nil
	}
	if scope.In == "" {
		return view.contextPath, nil
	}
	for i := range view.Boards {
		if view.Boards[i].Path == scope.In {
			return scope.In, nil
		}
	}
	return "", fmt.Errorf("unknown track %q", scope.In)
}

// Under returns the board at basePath and every board below it, in walk order.
func (view *TreeView) Under(basePath string) []BoardView {
	var out []BoardView
	for i := range view.Boards {
		if within(view.Boards[i].Path, basePath) {
			out = append(out, view.Boards[i])
		}
	}
	return out
}

func within(path, base string) bool {
	return base == "." || path == base || strings.HasPrefix(path, base+"/")
}
