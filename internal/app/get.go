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
	return a.taskInfo(root, path)
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
// returns it with the truthful post-claim status, so parallel callers each get
// a distinct task.
func (a App) GetNext(cwd, in string, claim bool) (*TaskInfo, error) {
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	info, err := a.nextActionable(root, cwd, in)
	if err != nil || info == nil || !claim {
		return info, err
	}
	return a.claim(root, cwd, in)
}

// claim marks the next actionable task in-progress under the tree lock and
// returns it with the truthful post-claim status. Re-scanning under the lock —
// the authoritative pass — is what makes parallel claims each pick a distinct
// task; the caller's earlier unlocked scan only decides whether to lock at all,
// so a claim with nothing to do leaves no lock-file side effect.
func (a App) claim(root, cwd, in string) (*TaskInfo, error) {
	unlock, err := a.lock(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	info, err := a.nextActionable(root, cwd, in)
	if err != nil || info == nil {
		return info, err
	}
	if err := a.setStatusLocked(info.Path, info.ID, task.StatusInProgress, false); err != nil {
		return nil, err
	}
	info.Status = task.StatusInProgress
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
	ready, err := a.depsDone(root, t.BlockedBy)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, nil
	}
	info := buildTaskInfo(root, path, content, t, a.mtime(path))
	return &info, nil
}

// depsDone reports whether every id in deps resolves to a done task.
func (a App) depsDone(root string, deps []string) (bool, error) {
	for _, id := range deps {
		done, err := a.depDone(root, id)
		if err != nil {
			return false, err
		}
		if !done {
			return false, nil
		}
	}
	return true, nil
}

// depDone reports whether dependency id counts as satisfied: a done open task
// or any archived task. An id that resolves nowhere counts as not done,
// leaving the dependent task blocked.
func (a App) depDone(root, id string) (bool, error) {
	path, err := tree.ResolveTask(a.fs, root, id)
	if err != nil {
		if errors.Is(err, tree.ErrArchived) {
			return true, nil
		}
		return false, nil
	}
	t, _, err := a.readTask(path)
	if err != nil {
		return false, err
	}
	return t.Status == task.StatusDone, nil
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
