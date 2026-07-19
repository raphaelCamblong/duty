package tests

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raphaelCamblong/duty/internal/app"
	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
)

// memTree builds a duty tree directly on a fsys.Mem, one primitive per fixture
// shape, so a test can craft any file/board (dis)agreement precisely.
type memTree struct {
	t    *testing.T
	fs   *fsys.Mem
	root string
}

func newMemTree(t *testing.T) *memTree {
	t.Helper()
	mem := fsys.NewMem()
	root := "/duty"
	if err := mem.MkdirAll(root); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	tr := &memTree{t: t, fs: mem, root: root}
	tr.write(filepath.Join(root, names.ConfigFile), nil)
	tr.write(filepath.Join(root, names.BoardFile), board.Render("Root"))
	return tr
}

func (tr *memTree) write(path string, content []byte) {
	tr.t.Helper()
	if err := tr.fs.WriteFile(path, content); err != nil {
		tr.t.Fatalf("write %s: %v", path, err)
	}
}

// taskFile writes a task file with the given status and returns its filename.
func (tr *memTree) taskFile(dir, id, title, status string, blockedBy []string) string {
	tr.t.Helper()
	content := task.Render(id, title, blockedBy)
	if status != task.StatusTodo {
		out, err := task.SetStatus(content, status)
		if err != nil {
			tr.t.Fatalf("set status %q: %v", status, err)
		}
		content = out
	}
	name := id + "-" + task.Slugify(title) + ".md"
	tr.write(filepath.Join(dir, name), content)
	return name
}

// row appends a board row to the default section.
func (tr *memTree) row(dir, id, file, title, status string) {
	tr.t.Helper()
	path := filepath.Join(dir, names.BoardFile)
	content, err := tr.fs.ReadFile(path)
	if err != nil {
		tr.t.Fatalf("read board %s: %v", path, err)
	}
	updated, err := board.AddRow(content, board.DefaultSection, board.Row{ID: id, File: file, Title: title, Status: status})
	if err != nil {
		tr.t.Fatalf("add row %s: %v", id, err)
	}
	tr.write(path, updated)
}

// synced writes a task file and a board row that agree.
func (tr *memTree) synced(dir, id, title, status string, blockedBy []string) {
	name := tr.taskFile(dir, id, title, status, blockedBy)
	tr.row(dir, id, name, title, status)
}

// archived plants an archived task file under dir/archive.
func (tr *memTree) archived(dir, id, title, status string) {
	tr.t.Helper()
	archiveDir := filepath.Join(dir, names.ArchiveDir)
	if err := tr.fs.MkdirAll(archiveDir); err != nil {
		tr.t.Fatalf("mkdir archive: %v", err)
	}
	tr.taskFile(archiveDir, id, title, status, nil)
}

// track creates a sub-board under parentDir and returns its directory.
func (tr *memTree) track(parentDir, name, title string) string {
	tr.t.Helper()
	dir := filepath.Join(parentDir, name)
	if err := tr.fs.MkdirAll(dir); err != nil {
		tr.t.Fatalf("mkdir track %s: %v", name, err)
	}
	tr.write(filepath.Join(dir, names.BoardFile), board.Render(title))
	return dir
}

func (tr *memTree) load(cwd string, opts app.LoadOptions) *app.TreeView {
	tr.t.Helper()
	view, err := app.New(tr.fs).Load(cwd, opts)
	if err != nil {
		tr.t.Fatalf("load %s: %v", cwd, err)
	}
	return view
}

func findTask(view *app.TreeView, id string) (app.TaskView, bool) {
	for _, boardView := range view.Boards {
		for _, section := range boardView.Sections {
			for _, item := range section.Tasks {
				if item.ID == id {
					return item, true
				}
			}
		}
	}
	return app.TaskView{}, false
}

func taskOf(t *testing.T, view *app.TreeView, id string) app.TaskView {
	t.Helper()
	item, ok := findTask(view, id)
	if !ok {
		t.Fatalf("task %s not in view", id)
	}
	return item
}

func countTask(view *app.TreeView, id string) int {
	count := 0
	for _, boardView := range view.Boards {
		for _, section := range boardView.Sections {
			for _, item := range section.Tasks {
				if item.ID == id {
					count++
				}
			}
		}
	}
	return count
}

func defaultSection(boardView app.BoardView) app.SectionView {
	for _, section := range boardView.Sections {
		if section.Name == board.DefaultSection {
			return section
		}
	}
	return app.SectionView{}
}

func depStatusOf(item app.TaskView, id string) string {
	for _, dep := range item.Deps {
		if dep.ID == id {
			return dep.Status
		}
	}
	return ""
}

func boardPaths(boards []app.BoardView) []string {
	paths := make([]string, len(boards))
	for i, boardView := range boards {
		paths[i] = boardView.Path
	}
	return paths
}

func TestLoadJoinsSyncedTask(t *testing.T) {
	tr := newMemTree(t)
	tr.synced(tr.root, "T-01", "First task", task.StatusTodo, nil)

	view := tr.load(tr.root, app.LoadOptions{})

	if len(view.Boards) != 1 {
		t.Fatalf("boards = %d, want 1", len(view.Boards))
	}
	root := view.Boards[0]
	if root.Path != "." || root.Title != "Root" {
		t.Errorf("root board = %q / %q, want . / Root", root.Path, root.Title)
	}
	item := taskOf(t, view, "T-01")
	if item.Drift != app.DriftNone {
		t.Errorf("drift = %v, want DriftNone", item.Drift)
	}
	if item.RowStatus != "" {
		t.Errorf("RowStatus = %q, want empty (in sync)", item.RowStatus)
	}
	if item.Status != task.StatusTodo || item.File != "T-01-first-task.md" {
		t.Errorf("task = %q / %q, want todo / T-01-first-task.md", item.Status, item.File)
	}
	if item.Path == "" || len(item.Content) == 0 {
		t.Errorf("file truth missing: path=%q content=%d bytes", item.Path, len(item.Content))
	}
	if got, ok := view.Task("T-01"); !ok || got.ID != "T-01" {
		t.Errorf("Task(T-01) = %v / %v, want the loaded task", got, ok)
	}
}

func TestLoadDriftClasses(t *testing.T) {
	tr := newMemTree(t)
	tr.synced(tr.root, "T-01", "Synced", task.StatusTodo, nil)
	// status drift: the file says done, the row still says todo.
	driftName := tr.taskFile(tr.root, "T-02", "Drifted", task.StatusDone, nil)
	tr.row(tr.root, "T-02", driftName, "Drifted", task.StatusTodo)
	// no row: a file with no board row.
	tr.taskFile(tr.root, "T-03", "Rowless", task.StatusTodo, nil)
	// no file: a board row pointing at a file that does not exist.
	tr.row(tr.root, "T-04", "T-04-ghost.md", "Ghost", task.StatusBlocked)
	// bad file: an unparsable file the board indexes.
	badRaw := []byte("not valid frontmatter\n")
	tr.write(filepath.Join(tr.root, "T-05-bad.md"), badRaw)
	tr.row(tr.root, "T-05", "T-05-bad.md", "Bad", task.StatusTodo)

	view := tr.load(tr.root, app.LoadOptions{})

	t.Run("none", func(t *testing.T) {
		item := taskOf(t, view, "T-01")
		if item.Drift != app.DriftNone || item.RowStatus != "" {
			t.Errorf("drift=%v row=%q, want DriftNone / empty", item.Drift, item.RowStatus)
		}
	})
	t.Run("status", func(t *testing.T) {
		item := taskOf(t, view, "T-02")
		if item.Drift != app.DriftStatus || item.Status != task.StatusDone || item.RowStatus != task.StatusTodo {
			t.Errorf("drift=%v status=%q row=%q, want DriftStatus / done (file wins) / todo", item.Drift, item.Status, item.RowStatus)
		}
	})
	t.Run("no row", func(t *testing.T) {
		item := taskOf(t, view, "T-03")
		if item.Drift != app.DriftNoRow || item.Path == "" {
			t.Errorf("drift=%v path=%q, want DriftNoRow with file truth", item.Drift, item.Path)
		}
	})
	t.Run("no file", func(t *testing.T) {
		item := taskOf(t, view, "T-04")
		if item.Drift != app.DriftNoFile || item.Path != "" {
			t.Errorf("drift=%v path=%q, want DriftNoFile with no path", item.Drift, item.Path)
		}
		if item.Status != task.StatusBlocked || item.RowStatus != task.StatusBlocked {
			t.Errorf("status=%q row=%q, want blocked / blocked (board truth)", item.Status, item.RowStatus)
		}
	})
	t.Run("bad file", func(t *testing.T) {
		item := taskOf(t, view, "T-05")
		if item.Drift != app.DriftBadFile || !bytes.Equal(item.Content, badRaw) {
			t.Errorf("drift=%v content=%q, want DriftBadFile carrying the raw bytes", item.Drift, item.Content)
		}
		if item.Title != "Bad" || item.Status != task.StatusTodo || item.Path == "" {
			t.Errorf("identity=%q/%q path=%q, want Bad / todo (board truth) with a path", item.Title, item.Status, item.Path)
		}
	})
}

func TestLoadStrayRule(t *testing.T) {
	t.Run("rowless files sort by filename into the default section", func(t *testing.T) {
		tr := newMemTree(t)
		tr.synced(tr.root, "T-01", "Has row", task.StatusTodo, nil)
		// Added out of filename order to prove the sort is by name, not add order.
		tr.taskFile(tr.root, "T-03", "Zeta stray", task.StatusTodo, nil)
		tr.taskFile(tr.root, "T-02", "Alpha stray", task.StatusTodo, nil)

		view := tr.load(tr.root, app.LoadOptions{})
		tasks := defaultSection(view.Boards[0]).Tasks
		if len(tasks) != 3 {
			t.Fatalf("default section = %d tasks, want 3 (row + 2 strays)", len(tasks))
		}
		if tasks[1].ID != "T-02" || tasks[2].ID != "T-03" {
			t.Errorf("stray order = %q, %q, want T-02 then T-03 by filename", tasks[1].ID, tasks[2].ID)
		}
		if tasks[1].Drift != app.DriftNoRow || tasks[2].Drift != app.DriftNoRow {
			t.Errorf("strays not flagged DriftNoRow: %v, %v", tasks[1].Drift, tasks[2].Drift)
		}
	})

	t.Run("a duplicate row renders the task once", func(t *testing.T) {
		tr := newMemTree(t)
		name := tr.taskFile(tr.root, "T-01", "Dup", task.StatusTodo, nil)
		tr.row(tr.root, "T-01", name, "Dup", task.StatusTodo)
		tr.row(tr.root, "T-01", name, "Dup", task.StatusTodo)

		view := tr.load(tr.root, app.LoadOptions{})
		if got := countTask(view, "T-01"); got != 1 {
			t.Errorf("T-01 rendered %d times, want 1 (first row wins)", got)
		}
	})

	t.Run("the default section is created last when the index has none", func(t *testing.T) {
		tr := newMemTree(t)
		custom := "# Root\n\n## Later\n\n| Task | Title | Status |\n|------|-------|--------|\n\n" +
			"Completed tasks (0) archived: [archive/](archive/).\n"
		tr.write(filepath.Join(tr.root, names.BoardFile), []byte(custom))
		tr.taskFile(tr.root, "T-01", "Stray", task.StatusTodo, nil)

		view := tr.load(tr.root, app.LoadOptions{})
		sections := view.Boards[0].Sections
		last := sections[len(sections)-1]
		if last.Name != board.DefaultSection {
			t.Fatalf("last section = %q, want %q created for the stray", last.Name, board.DefaultSection)
		}
		if len(last.Tasks) != 1 || last.Tasks[0].ID != "T-01" {
			t.Errorf("default section = %+v, want one stray T-01", last.Tasks)
		}
	})
}

func TestLoadDepOracle(t *testing.T) {
	tr := newMemTree(t)
	tr.synced(tr.root, "T-10", "Done dep", task.StatusDone, nil)
	tr.synced(tr.root, "T-01", "Waits on done", task.StatusTodo, []string{"T-10"})
	tr.archived(tr.root, "T-11", "Archived dep", task.StatusDone)
	tr.synced(tr.root, "T-02", "Waits on archived", task.StatusTodo, []string{"T-11"})
	tr.synced(tr.root, "T-12", "Backlog dep", task.StatusBacklog, nil)
	tr.synced(tr.root, "T-03", "Waits on backlog", task.StatusTodo, []string{"T-12"})
	tr.synced(tr.root, "T-04", "Waits on missing", task.StatusTodo, []string{"T-99"})
	// board-only row: an id with a row but no file has no file truth.
	tr.row(tr.root, "T-13", "T-13-ghost.md", "Ghost dep", task.StatusDone)
	tr.synced(tr.root, "T-05", "Waits on board-only", task.StatusTodo, []string{"T-13"})

	// Archive off proves the oracle answers "archived" from filenames alone.
	view := tr.load(tr.root, app.LoadOptions{})

	cases := []struct {
		id, dep, status string
		met             bool
	}{
		{"T-01", "T-10", "done", true},
		{"T-02", "T-11", "archived", true},
		{"T-03", "T-12", task.StatusBacklog, false},
		{"T-04", "T-99", "missing", false},
		{"T-05", "T-13", "missing", false},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			item := taskOf(t, view, tc.id)
			if got := depStatusOf(item, tc.dep); got != tc.status {
				t.Errorf("dep %s status = %q, want %q", tc.dep, got, tc.status)
			}
			waiting := len(item.Waits) > 0
			if waiting == tc.met {
				t.Errorf("waits = %v, want met=%v (so waiting=%v)", item.Waits, tc.met, !tc.met)
			}
			if !tc.met && (len(item.Waits) != 1 || item.Waits[0] != tc.dep) {
				t.Errorf("waits = %v, want exactly [%s]", item.Waits, tc.dep)
			}
		})
	}
}

func TestLoadArchiveReadsGatedByOption(t *testing.T) {
	tr := newMemTree(t)
	tr.synced(tr.root, "T-01", "Open", task.StatusTodo, nil)
	tr.archived(tr.root, "T-02", "Archived one", task.StatusDone)
	tr.archived(tr.root, "T-03", "Archived two", task.StatusDone)

	off := &countingFS{FS: tr.fs}
	view, err := app.New(off).Load(tr.root, app.LoadOptions{})
	if err != nil {
		t.Fatalf("load off: %v", err)
	}
	if off.archiveReads != 0 {
		t.Errorf("archive content reads while off = %d, want 0 (cost discipline)", off.archiveReads)
	}
	root := view.Boards[0]
	if root.ArchivedCount != 2 {
		t.Errorf("ArchivedCount = %d, want 2 (tallied from the listing)", root.ArchivedCount)
	}
	if len(root.Archived) != 0 {
		t.Errorf("Archived rows = %d while off, want 0", len(root.Archived))
	}

	on := &countingFS{FS: tr.fs}
	loaded, err := app.New(on).Load(tr.root, app.LoadOptions{Archive: true})
	if err != nil {
		t.Fatalf("load on: %v", err)
	}
	if on.archiveReads != 2 {
		t.Errorf("archive content reads while on = %d, want 2 (T-02 and T-03)", on.archiveReads)
	}
	rows := loaded.Boards[0].Archived
	if len(rows) != 2 {
		t.Fatalf("Archived rows = %d while on, want 2", len(rows))
	}
	for _, item := range rows {
		if len(item.Content) == 0 {
			t.Errorf("archived %s carries no content", item.ID)
		}
	}
}

func TestScopeBase(t *testing.T) {
	tr := newMemTree(t)
	sub := tr.track(tr.root, "sub", "Sub track")

	t.Run("dot resolves to the root", func(t *testing.T) {
		base, err := tr.load(tr.root, app.LoadOptions{}).ScopeBase(app.Scope{In: "."})
		if err != nil || base != "." {
			t.Errorf("ScopeBase(.) = %q, %v, want . / nil", base, err)
		}
	})
	t.Run("empty resolves to the context board at the root", func(t *testing.T) {
		base, err := tr.load(tr.root, app.LoadOptions{}).ScopeBase(app.Scope{In: ""})
		if err != nil || base != "." {
			t.Errorf("ScopeBase() from root = %q, %v, want . / nil", base, err)
		}
	})
	t.Run("empty resolves to the context board inside a track", func(t *testing.T) {
		base, err := tr.load(sub, app.LoadOptions{}).ScopeBase(app.Scope{In: ""})
		if err != nil || base != "sub" {
			t.Errorf("ScopeBase() from sub = %q, %v, want sub / nil", base, err)
		}
	})
	t.Run("a named track resolves to itself", func(t *testing.T) {
		base, err := tr.load(tr.root, app.LoadOptions{}).ScopeBase(app.Scope{In: "sub"})
		if err != nil || base != "sub" {
			t.Errorf("ScopeBase(sub) = %q, %v, want sub / nil", base, err)
		}
	})
	t.Run("an unknown track errors", func(t *testing.T) {
		_, err := tr.load(tr.root, app.LoadOptions{}).ScopeBase(app.Scope{In: "nope"})
		if err == nil || !strings.Contains(err.Error(), `unknown track "nope"`) {
			t.Errorf("ScopeBase(nope) error = %v, want unknown track \"nope\"", err)
		}
	})
}

func TestUnderSubtreeOrder(t *testing.T) {
	tr := newMemTree(t)
	trackA := tr.track(tr.root, "a", "A")
	tr.track(trackA, "b", "B")
	tr.track(tr.root, "c", "C")

	view := tr.load(tr.root, app.LoadOptions{})

	if got := boardPaths(view.Under(".")); !equalStrings(got, []string{".", "a", "a/b", "c"}) {
		t.Errorf("Under(.) = %v, want . a a/b c in walk order", got)
	}
	if got := boardPaths(view.Under("a")); !equalStrings(got, []string{"a", "a/b"}) {
		t.Errorf("Under(a) = %v, want a a/b", got)
	}
	if got := boardPaths(view.Under("c")); !equalStrings(got, []string{"c"}) {
		t.Errorf("Under(c) = %v, want c", got)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
