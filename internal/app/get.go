package app

import (
	"path/filepath"
	"time"

	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// TaskInfo is one task's metadata read from its file — its track path and
// gate progress included, its body excluded. It is what GetTask and GetNext
// return.
type TaskInfo struct {
	ID         string
	Title      string
	Status     string
	Track      string // slash path of the task's board relative to the tree root, "." for the root
	BlockedBy  []string
	GatesDone  int
	GatesTotal int
	Path       string    // absolute path of the task file
	UpdatedAt  time.Time // file modification time
	ClaimedBy  string    // agent holding an in-progress task, "" when unclaimed
	Deps       []Dep     // blocked-by ids paired with their computed status
}

// Dep is one blocked-by prerequisite with the status duty computes for it:
// a lifecycle status for an open task, "archived" for an archived one, or
// "missing" when the id resolves to no task.
type Dep struct {
	ID     string
	Status string
}

// TrackInfo is one board's line for GetTracks: per-status counts of its own
// (directly-filed, non-recursive) tasks, plus its archived count.
type TrackInfo struct {
	Path       string
	Title      string
	Todo       int
	InProgress int
	Done       int
	Blocked    int
	Backlog    int
	Archived   int
}

// GetTask returns the metadata — never the body — of the open task id resolves
// to, projected from the loaded tree.
func (a App) GetTask(cwd, id string) (TaskInfo, error) {
	view, err := a.Load(cwd, LoadOptions{})
	if err != nil {
		return TaskInfo{}, err
	}
	found, ok := view.Task(id)
	if !ok {
		return TaskInfo{}, view.resolveErr(id)
	}
	return taskInfoFromView(view.Root, *found), nil
}

// Body returns the whole markdown body below the frontmatter of the task id
// resolves to, verbatim. Archived ids are read-only and rejected.
func (a App) Body(cwd, id string) (string, error) {
	_, path, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return "", err
	}
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(task.Body(content)), nil
}

// GetTracks returns one TrackInfo per board in scope and below (that board as
// "."), each counting its own directly-filed tasks by status.
func (a App) GetTracks(scope Scope) ([]TrackInfo, error) {
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
	out := make([]TrackInfo, 0, len(boards))
	for i := range boards {
		out = append(out, trackInfoOf(baseDir, boards[i]))
	}
	return out, nil
}

func trackInfoOf(baseDir string, bv BoardView) TrackInfo {
	info := TrackInfo{Path: relBoard(baseDir, bv.Dir), Title: bv.Title, Archived: bv.ArchivedCount}
	for _, tv := range bv.tasks() {
		countStatus(&info, tv)
	}
	return info
}

func countStatus(info *TrackInfo, tv TaskView) {
	if !tv.hasFileTruth() {
		return
	}
	switch tv.Status {
	case task.StatusTodo:
		info.Todo++
	case task.StatusInProgress:
		info.InProgress++
	case task.StatusDone:
		info.Done++
	case task.StatusBlocked:
		info.Blocked++
	case task.StatusBacklog:
		info.Backlog++
	}
}

// GetNext returns the first actionable task in scope and below, or nil; with
// claim it atomically marks that task in-progress as as before returning it.
func (a App) GetNext(scope Scope, claim bool, as string) (*TaskInfo, error) {
	view, err := a.Load(scope.Cwd, LoadOptions{})
	if err != nil {
		return nil, err
	}
	winner, err := nextIn(view, scope)
	if err != nil || winner == nil {
		return nil, err
	}
	if claim {
		return a.claimNext(scope, as)
	}
	info := taskInfoFromView(view.Root, *winner)
	return &info, nil
}

// claimNext re-loads under the tree lock — the authoritative pass that makes
// parallel claims each pick a distinct task — and marks that task in-progress.
func (a App) claimNext(scope Scope, as string) (*TaskInfo, error) {
	root, err := tree.FindRoot(a.fs, scope.Cwd)
	if err != nil {
		return nil, err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	view, err := a.Load(scope.Cwd, LoadOptions{})
	if err != nil {
		return nil, err
	}
	winner, err := nextIn(view, scope)
	if err != nil || winner == nil {
		return nil, err
	}
	if err := a.setStatusLocked(winner.Path, StatusChange{ID: winner.ID, Status: task.StatusInProgress, As: as}); err != nil {
		return nil, err
	}
	info := taskInfoFromView(view.Root, *winner)
	info.Status = task.StatusInProgress
	info.ClaimedBy = as
	return &info, nil
}

// nextIn returns the first actionable task in scope's board and below, in
// flattened board order — the winner get next reports and claim marks.
func nextIn(view *TreeView, scope Scope) (*TaskView, error) {
	base, err := view.ScopeBase(scope)
	if err != nil {
		return nil, err
	}
	boards := view.Under(base)
	for i := range boards {
		if found, ok := firstActionable(boards[i]); ok {
			return &found, nil
		}
	}
	return nil, nil
}

func firstActionable(bv BoardView) (TaskView, bool) {
	for _, tv := range bv.tasks() {
		if actionable(tv) {
			return tv, true
		}
	}
	return TaskView{}, false
}

// actionable reports whether a task is ready to pick up: a file-truth todo with
// every dependency met. No-file and bad-file rows carry no file truth.
func actionable(tv TaskView) bool {
	return tv.hasFileTruth() && tv.Status == task.StatusTodo && len(tv.Waits) == 0
}

// taskInfoFromView projects one task's metadata straight from its loaded view —
// Deps and gate counts already computed by Load, so no file is re-read.
func taskInfoFromView(root string, tv TaskView) TaskInfo {
	return TaskInfo{
		ID:         tv.ID,
		Title:      tv.Title,
		Status:     tv.Status,
		Track:      relBoard(root, filepath.Dir(tv.Path)),
		BlockedBy:  tv.BlockedBy,
		GatesDone:  tv.GatesDone,
		GatesTotal: tv.GatesTotal,
		Path:       tv.Path,
		UpdatedAt:  tv.UpdatedAt,
		ClaimedBy:  tv.ClaimedBy,
		Deps:       tv.Deps,
	}
}

// statusArchived and statusMissing are the computed dependency statuses for a
// blocked-by id that resolves to an archived task, or to no task at all;
// neither is a lifecycle status, so neither is ever written to a file.
const (
	statusArchived = "archived"
	statusMissing  = "missing"
)

// UnmetDeps returns the deps statusOf reports as neither done nor archived —
// the one wait-state rule behind get next, get tasks' waits, and the TUI.
func UnmetDeps(deps []string, statusOf func(id string) (string, error)) ([]string, error) {
	var unmet []string
	for _, id := range deps {
		status, err := statusOf(id)
		if err != nil {
			return nil, err
		}
		if !depMet(status) {
			unmet = append(unmet, id)
		}
	}
	return unmet, nil
}

func depMet(status string) bool {
	return status == task.StatusDone || status == statusArchived
}

// relBoard returns dir's slash path relative to root, "." for root itself.
func relBoard(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}
