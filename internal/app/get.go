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

// TrackInfo is one board's line for GetTracks: its path, title, per-status
// own-task counts, and archived count.
type TrackInfo struct {
	Path       string
	Title      string
	Todo       int
	InProgress int
	Done       int
	Blocked    int
	Archived   int
}

// GetTask returns the metadata of the open task id resolves to anywhere in
// the tree containing cwd. It reads the frontmatter and gates, never the body.
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

// GetTracks returns one TrackInfo per board in the board in — a root-relative
// track path, or the board containing cwd when empty — and every board below
// it, that board included as ".". Counts tally the board's own
// (directly-filed) tasks by status; Archived counts its archive/.
func (a App) GetTracks(cwd, in string) ([]TrackInfo, error) {
	boardDir, boards, err := a.walkBoards(cwd, in)
	if err != nil {
		return nil, err
	}
	out := make([]TrackInfo, 0, len(boards))
	for _, b := range boards {
		info, err := a.trackInfo(boardDir, b)
		if err != nil {
			return nil, err
		}
		out = append(out, info)
	}
	return out, nil
}

// GetNext returns the first actionable task: scanning the board in — a
// root-relative track path, or the board containing cwd when empty — and
// every board below it in scan order, and within each its rows in board
// order, it returns the first todo whose blocked-by are all done — archived
// dependencies count as done. It returns nil when nothing is actionable. With
// claim set it atomically marks that task in-progress under the tree lock and
// returns it with the truthful post-claim status and as as its claimer, so
// parallel callers each get a distinct task.
func (a App) GetNext(cwd, in string, claim bool, as string) (*TaskInfo, error) {
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	info, err := a.nextActionable(root, cwd, in)
	if err != nil || info == nil {
		return info, err
	}
	if claim {
		info, err = a.claim(root, cwd, in, as)
		if err != nil || info == nil {
			return info, err
		}
	}
	if err := a.fillDeps(root, info); err != nil {
		return nil, err
	}
	return info, nil
}

// claim marks the next actionable task in-progress under the tree lock and
// returns it with the truthful post-claim status. Re-scanning under the lock —
// the authoritative pass — is what makes parallel claims each pick a distinct
// task; the caller's earlier unlocked scan only decides whether to lock at all,
// so a claim with nothing to do leaves no lock-file side effect.
func (a App) claim(root, cwd, in, as string) (*TaskInfo, error) {
	unlock, err := a.lock(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	info, err := a.nextActionable(root, cwd, in)
	if err != nil || info == nil {
		return info, err
	}
	if err := a.setStatusLocked(info.Path, info.ID, task.StatusInProgress, false, as); err != nil {
		return nil, err
	}
	info.Status = task.StatusInProgress
	info.ClaimedBy = as
	return info, nil
}

// nextActionable scans the board in and every board below it and returns the
// first actionable task, or nil when none is ready.
func (a App) nextActionable(root, cwd, in string) (*TaskInfo, error) {
	_, boards, err := a.walkBoards(cwd, in)
	if err != nil {
		return nil, err
	}
	for _, b := range boards {
		info, err := a.nextInBoard(root, b)
		if err != nil {
			return nil, err
		}
		if info != nil {
			return info, nil
		}
	}
	return nil, nil
}

// taskInfo reads the task file at path and assembles its TaskInfo.
func (a App) taskInfo(root, path string) (TaskInfo, error) {
	t, content, err := a.readTask(path)
	if err != nil {
		return TaskInfo{}, err
	}
	return buildTaskInfo(root, path, content, t, a.mtime(path)), nil
}

// buildTaskInfo assembles a TaskInfo from an already-read task file.
func buildTaskInfo(root, path string, content []byte, t task.Task, updated time.Time) TaskInfo {
	done, total := task.CountGates(content)
	return TaskInfo{
		ID:         t.ID,
		Title:      t.Title,
		Status:     t.Status,
		Track:      relBoard(root, filepath.Dir(path)),
		BlockedBy:  t.BlockedBy,
		GatesDone:  done,
		GatesTotal: total,
		Path:       path,
		UpdatedAt:  updated,
		ClaimedBy:  t.ClaimedBy,
	}
}

// mtime returns the modification time of the file at path, the zero time when
// it cannot be stat'd. Age is display metadata, so a stat miss degrades to no
// age rather than failing the read.
func (a App) mtime(path string) time.Time {
	info, err := a.fs.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// trackInfo assembles one board's TrackInfo, its path taken relative to root.
func (a App) trackInfo(root, b string) (TrackInfo, error) {
	index, err := a.fs.ReadFile(filepath.Join(b, names.BoardFile))
	if err != nil {
		return TrackInfo{}, err
	}
	info := TrackInfo{Path: relBoard(root, b), Title: board.TitleOr(index, filepath.Base(b))}
	statuses, err := a.taskStatuses(b)
	if err != nil {
		return TrackInfo{}, err
	}
	for _, s := range statuses {
		switch s {
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
	info.Archived, err = a.archivedCount(filepath.Join(b, names.ArchiveDir))
	if err != nil {
		return TrackInfo{}, err
	}
	return info, nil
}

// nextInBoard returns the first actionable task filed directly in board b,
// scanning its rows in board order, or nil when b holds none.
func (a App) nextInBoard(root, b string) (*TaskInfo, error) {
	index, err := a.fs.ReadFile(filepath.Join(b, names.BoardFile))
	if err != nil {
		return nil, err
	}
	for _, sec := range board.Sections(index) {
		for _, r := range sec.Rows {
			info, err := a.actionable(root, b, r.File)
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

// actionable returns the TaskInfo for filename in board b when its file is a
// todo whose blocked-by are all done, else nil. A row pointing at a missing
// file is skipped, not an error — the board can drift ahead of the files.
func (a App) actionable(root, b, filename string) (*TaskInfo, error) {
	path := filepath.Join(b, filename)
	t, content, err := a.readTask(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if t.Status != task.StatusTodo {
		return nil, nil
	}
	waits, err := a.unmetDeps(root, t.BlockedBy)
	if err != nil {
		return nil, err
	}
	if len(waits) > 0 {
		return nil, nil
	}
	info := buildTaskInfo(root, path, content, t, a.mtime(path))
	return &info, nil
}

// statusArchived and statusMissing are the computed dependency statuses for a
// blocked-by id that resolves to an archived task, or to no task at all;
// neither is a lifecycle status, so neither is ever written to a file.
const (
	statusArchived = "archived"
	statusMissing  = "missing"
)

// UnmetDeps returns the ids in deps that are not yet satisfied, each dep's
// status read through statusOf: a dep counts as met when its status is done or
// archived, unmet otherwise (a task still in flight, or one statusOf reports as
// missing). It is the single wait-state rule behind get next's actionable walk,
// get tasks' waits column, and the TUI's wait annotation.
func UnmetDeps(deps []string, statusOf func(id string) (string, error)) ([]string, error) {
	var unmet []string
	for _, id := range deps {
		s, err := statusOf(id)
		if err != nil {
			return nil, err
		}
		if !depMet(s) {
			unmet = append(unmet, id)
		}
	}
	return unmet, nil
}

// depMet reports whether a computed dependency status counts as satisfied:
// done, or archived — archived prerequisites are treated as done.
func depMet(status string) bool {
	return status == task.StatusDone || status == statusArchived
}

// unmetDeps returns deps' unmet ids, reading each dep's status from the files.
func (a App) unmetDeps(root string, deps []string) ([]string, error) {
	return UnmetDeps(deps, func(id string) (string, error) {
		return a.depStatus(root, id)
	})
}

// depStatuses pairs each blocked-by id with its computed status, in order.
func (a App) depStatuses(root string, deps []string) ([]Dep, error) {
	out := make([]Dep, 0, len(deps))
	for _, id := range deps {
		s, err := a.depStatus(root, id)
		if err != nil {
			return nil, err
		}
		out = append(out, Dep{ID: id, Status: s})
	}
	return out, nil
}

// fillDeps resolves info's blocked-by ids to their statuses and stores them.
func (a App) fillDeps(root string, info *TaskInfo) error {
	deps, err := a.depStatuses(root, info.BlockedBy)
	if err != nil {
		return err
	}
	info.Deps = deps
	return nil
}

// depStatus resolves dependency id to the status duty computes for it: the open
// task's lifecycle status, statusArchived for an archived task, or
// statusMissing when the id resolves to no task. It is the one source the wait
// computation and get task's annotations both read.
func (a App) depStatus(root, id string) (string, error) {
	path, err := tree.ResolveTask(a.fs, root, id)
	if err != nil {
		if errors.Is(err, tree.ErrArchived) {
			return statusArchived, nil
		}
		return statusMissing, nil
	}
	t, _, err := a.readTask(path)
	if err != nil {
		return "", err
	}
	return t.Status, nil
}

// taskStatuses returns the file status of every task filed directly in dir.
func (a App) taskStatuses(dir string) ([]string, error) {
	files, err := tree.TaskFileNames(a.fs, dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(files))
	for _, name := range files {
		t, _, err := a.readTask(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		out = append(out, t.Status)
	}
	return out, nil
}

// archivedCount counts the task files directly in dir, returning 0 when dir
// does not exist — a board with nothing archived may lack the folder.
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
