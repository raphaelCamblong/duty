package app

import (
	"errors"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
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
	Archived   int
}

// GetTask returns the metadata — never the body — of the open task id
// resolves to.
func (a App) GetTask(cwd, id string) (TaskInfo, error) {
	root, path, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return TaskInfo{}, err
	}
	info, err := a.taskInfo(root, path)
	if err != nil {
		return TaskInfo{}, err
	}
	if err := a.fillDeps(root, &info); err != nil {
		return TaskInfo{}, err
	}
	return info, nil
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
	boardDir, boards, err := a.walkBoards(scope)
	if err != nil {
		return nil, err
	}
	out := make([]TrackInfo, 0, len(boards))
	for _, dir := range boards {
		info, err := a.trackInfo(boardDir, dir)
		if err != nil {
			return nil, err
		}
		out = append(out, info)
	}
	return out, nil
}

// GetNext returns the first actionable task in scope and below, or nil; with
// claim it atomically marks that task in-progress as as before returning it.
func (a App) GetNext(scope Scope, claim bool, as string) (*TaskInfo, error) {
	root, err := tree.FindRoot(a.fs, scope.Cwd)
	if err != nil {
		return nil, err
	}
	info, err := a.nextActionable(root, scope)
	if err != nil || info == nil {
		return info, err
	}
	if claim {
		info, err = a.claim(root, scope, as)
		if err != nil || info == nil {
			return info, err
		}
	}
	if err := a.fillDeps(root, info); err != nil {
		return nil, err
	}
	return info, nil
}

// claim re-scans under the tree lock — the authoritative pass that makes
// parallel claims each pick a distinct task — and marks that task in-progress.
func (a App) claim(root string, scope Scope, as string) (*TaskInfo, error) {
	unlock, err := a.lock(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	info, err := a.nextActionable(root, scope)
	if err != nil || info == nil {
		return info, err
	}
	if err := a.setStatusLocked(info.Path, StatusChange{ID: info.ID, Status: task.StatusInProgress, As: as}); err != nil {
		return nil, err
	}
	info.Status = task.StatusInProgress
	info.ClaimedBy = as
	return info, nil
}

func (a App) nextActionable(root string, scope Scope) (*TaskInfo, error) {
	_, boards, err := a.walkBoards(scope)
	if err != nil {
		return nil, err
	}
	for _, boardDir := range boards {
		info, err := a.nextInBoard(root, boardDir)
		if err != nil {
			return nil, err
		}
		if info != nil {
			return info, nil
		}
	}
	return nil, nil
}

func (a App) taskInfo(root, path string) (TaskInfo, error) {
	parsed, content, err := a.readTask(path)
	if err != nil {
		return TaskInfo{}, err
	}
	return buildTaskInfo(root, path, content, parsed, a.mtime(path)), nil
}

func buildTaskInfo(root, path string, content []byte, parsed task.Task, updated time.Time) TaskInfo {
	done, total := task.CountGates(content)
	return TaskInfo{
		ID:         parsed.ID,
		Title:      parsed.Title,
		Status:     parsed.Status,
		Track:      relBoard(root, filepath.Dir(path)),
		BlockedBy:  parsed.BlockedBy,
		GatesDone:  done,
		GatesTotal: total,
		Path:       path,
		UpdatedAt:  updated,
		ClaimedBy:  parsed.ClaimedBy,
	}
}

// mtime returns path's modification time, or the zero time on a stat miss —
// age is display metadata, so it degrades quietly rather than failing the read.
func (a App) mtime(path string) time.Time {
	info, err := a.fs.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// trackInfo assembles one board's TrackInfo, its path taken relative to
// listDir — the board the tracks listing started from.
func (a App) trackInfo(listDir, boardDir string) (TrackInfo, error) {
	index, err := a.fs.ReadFile(boardIndexPath(boardDir))
	if err != nil {
		return TrackInfo{}, err
	}
	info := TrackInfo{Path: relBoard(listDir, boardDir), Title: board.TitleOr(index, filepath.Base(boardDir))}
	statuses, err := a.taskStatuses(boardDir)
	if err != nil {
		return TrackInfo{}, err
	}
	for _, status := range statuses {
		switch status {
		case task.StatusTodo:
			info.Todo++
		case task.StatusInProgress:
			info.InProgress++
		case task.StatusDone:
			info.Done++
		case task.StatusBlocked:
			info.Blocked++
		}
	}
	info.Archived, err = a.archivedCount(filepath.Join(boardDir, names.ArchiveDir))
	if err != nil {
		return TrackInfo{}, err
	}
	return info, nil
}

func (a App) nextInBoard(root, boardDir string) (*TaskInfo, error) {
	index, err := a.fs.ReadFile(boardIndexPath(boardDir))
	if err != nil {
		return nil, err
	}
	for _, sec := range board.Sections(index) {
		for _, row := range sec.Rows {
			info, err := a.actionable(root, boardDir, row.File)
			if err != nil {
				return nil, err
			}
			if info != nil {
				return info, nil
			}
		}
	}
	return nil, nil
}

// actionable returns filename's TaskInfo when it's a todo with every blocked-by
// done, else nil; a missing file is skipped (the board can drift ahead), not an error.
func (a App) actionable(root, boardDir, filename string) (*TaskInfo, error) {
	path := filepath.Join(boardDir, filename)
	parsed, content, err := a.readTask(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if parsed.Status != task.StatusTodo {
		return nil, nil
	}
	waits, err := a.unmetDeps(root, parsed.BlockedBy)
	if err != nil {
		return nil, err
	}
	if len(waits) > 0 {
		return nil, nil
	}
	info := buildTaskInfo(root, path, content, parsed, a.mtime(path))
	return &info, nil
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

func (a App) unmetDeps(root string, deps []string) ([]string, error) {
	return UnmetDeps(deps, func(id string) (string, error) {
		return a.depStatus(root, id)
	})
}

func (a App) fillDeps(root string, info *TaskInfo) error {
	info.Deps = make([]Dep, 0, len(info.BlockedBy))
	for _, id := range info.BlockedBy {
		status, err := a.depStatus(root, id)
		if err != nil {
			return err
		}
		info.Deps = append(info.Deps, Dep{ID: id, Status: status})
	}
	return nil
}

// depStatus is the one source both wait computation and get task's dep
// annotations read for a blocked-by id's computed status.
func (a App) depStatus(root, id string) (string, error) {
	path, err := tree.ResolveTask(a.fs, root, id)
	if err != nil {
		if errors.Is(err, tree.ErrArchived) {
			return statusArchived, nil
		}
		return statusMissing, nil
	}
	parsed, _, err := a.readTask(path)
	if err != nil {
		return "", err
	}
	return parsed.Status, nil
}

func (a App) taskStatuses(dir string) ([]string, error) {
	_, tasks, err := a.tasksIn(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(tasks))
	for _, parsed := range tasks {
		out = append(out, parsed.Status)
	}
	return out, nil
}

// archivedCount counts dir's task files, or 0 when dir doesn't exist — a
// board with nothing archived may lack the folder.
func (a App) archivedCount(dir string) (int, error) {
	info, err := a.fs.Stat(dir)
	if err != nil || !info.IsDir() {
		return 0, nil
	}
	return a.countTaskFiles(dir)
}

// relBoard returns dir's slash path relative to root, "." for root itself.
func relBoard(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}
