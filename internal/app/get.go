package app

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

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
	Path       string // absolute path of the task file
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
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return TaskInfo{}, err
	}
	path, err := tree.ResolveTask(a.fs, root, id)
	if err != nil {
		return TaskInfo{}, err
	}
	return a.taskInfo(root, path)
}

// GetTracks returns one TrackInfo per board in the board containing cwd and
// every board below it — the current board included as ".". Counts tally the
// board's own (directly-filed) tasks by status; Archived counts its archive/.
func (a App) GetTracks(cwd string) ([]TrackInfo, error) {
	boardDir, err := tree.CurrentBoard(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	boards, err := tree.Boards(a.fs, boardDir)
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

// GetNext returns the first actionable task: scanning the board containing
// cwd and every board below it in scan order, and within each its rows in
// board order, it returns the first todo whose blocked-by are all done —
// archived dependencies count as done. It returns nil when nothing is
// actionable.
func (a App) GetNext(cwd string) (*TaskInfo, error) {
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	boardDir, err := tree.CurrentBoard(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	boards, err := tree.Boards(a.fs, boardDir)
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
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return TaskInfo{}, err
	}
	t, err := task.Parse(content)
	if err != nil {
		return TaskInfo{}, fmt.Errorf("%s: %w", path, err)
	}
	return buildTaskInfo(root, path, content, t), nil
}

// buildTaskInfo assembles a TaskInfo from an already-read task file.
func buildTaskInfo(root, path string, content []byte, t task.Task) TaskInfo {
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
	}
}

// trackInfo assembles one board's TrackInfo, its path taken relative to root.
func (a App) trackInfo(root, b string) (TrackInfo, error) {
	index, err := a.fs.ReadFile(filepath.Join(b, names.BoardFile))
	if err != nil {
		return TrackInfo{}, err
	}
	title := board.Title(index)
	if title == "" {
		title = filepath.Base(b)
	}
	info := TrackInfo{Path: relBoard(root, b), Title: title}
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
	content, err := a.fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	t, err := task.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
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
	info := buildTaskInfo(root, path, content, t)
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
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return false, err
	}
	t, err := task.Parse(content)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	return t.Status == task.StatusDone, nil
}

// taskStatuses returns the file status of every task filed directly in dir.
func (a App) taskStatuses(dir string) ([]string, error) {
	entries, err := a.fs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !tree.IsTaskFile(e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		content, err := a.fs.ReadFile(path)
		if err != nil {
			return nil, err
		}
		t, err := task.Parse(content)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
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
