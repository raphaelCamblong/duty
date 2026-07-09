package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/tree"
	"github.com/raphaelCamblong/duty/internal/tui"
)

// tuiTree builds a fixture tree via the CLI: root tasks T-01 (in-progress)
// and T-02 (todo), and a "backend" sub-board holding T-03 (done).
func tuiTree(t *testing.T) string {
	t.Helper()
	root := initDuty(t)
	mustDuty(t, root, "create", "Alpha task")
	mustDuty(t, root, "create", "Beta task")
	mustDuty(t, root, "board", "backend", "--title", "Backend")
	mustDuty(t, filepath.Join(root, "backend"), "create", "Gamma task")
	mustDuty(t, root, "status", "T-01", "in-progress")
	mustDuty(t, root, "status", "T-03", "done")
	return root
}

// mustDuty runs one duty command from dir and fails the test on any error.
func mustDuty(t *testing.T, dir string, args ...string) string {
	t.Helper()
	code, stdout, stderr := runDuty(t, dir, args...)
	if code != 0 {
		t.Fatalf("duty %v: code=%d stderr=%q", args, code, stderr)
	}
	return stdout
}

// mustScan scans root, failing the test on error.
func mustScan(t *testing.T, root string) tui.Snapshot {
	t.Helper()
	snap, err := tui.Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	return snap
}

// scanRow finds a task row by id in one board of a snapshot.
func scanRow(snap tui.Snapshot, boardPath, id string) (tui.Row, bool) {
	b, ok := snap.Boards[boardPath]
	if !ok {
		return tui.Row{}, false
	}
	for _, s := range b.Sections {
		for _, r := range s.Rows {
			if r.ID == id {
				return r, true
			}
		}
	}
	return tui.Row{}, false
}

// rewriteBoard applies edit to a board index on disk (drift fixtures are
// hand-made corruption, exactly what the TUI must surface).
func rewriteBoard(t *testing.T, dir string, edit func([]byte) ([]byte, error)) {
	t.Helper()
	path := filepath.Join(dir, tree.BoardFile)
	content, err := edit([]byte(readText(t, path)))
	if err != nil {
		t.Fatalf("edit board: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write board: %v", err)
	}
}

func TestScanViewModel(t *testing.T) {
	t.Run("sections and rows follow the board order", func(t *testing.T) {
		root := tuiTree(t)
		mustDuty(t, root, "link", "T-01", "Later")
		snap := mustScan(t, root)
		b := snap.Boards["."]
		var got []string
		for _, s := range b.Sections {
			for _, r := range s.Rows {
				got = append(got, s.Name+"/"+r.ID)
			}
		}
		want := []string{"Open tasks/T-02", "Later/T-01"}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("rows = %v, want %v", got, want)
		}
	})

	t.Run("row truth comes from the files", func(t *testing.T) {
		root := tuiTree(t)
		snap := mustScan(t, root)
		r, ok := scanRow(snap, ".", "T-01")
		if !ok {
			t.Fatal("T-01 not in snapshot")
		}
		if r.Status != "in-progress" || r.Title != "Alpha task" || r.Drift != "" {
			t.Errorf("row = %+v, want in-progress / Alpha task / no drift", r)
		}
		if !samePath(t, r.Path, filepath.Join(root, "T-01-alpha-task.md")) {
			t.Errorf("row path = %q", r.Path)
		}
	})

	t.Run("gate counts come from the checklist", func(t *testing.T) {
		root := tuiTree(t)
		path := filepath.Join(root, "T-01-alpha-task.md")
		content := strings.Replace(readText(t, path), "## Gates\n",
			"## Gates\n\n- [x] built\n- [ ] tested\n", 1)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		r, _ := scanRow(mustScan(t, root), ".", "T-01")
		if r.GatesDone != 1 || r.GatesTotal != 2 {
			t.Errorf("gates = %d/%d, want 1/2", r.GatesDone, r.GatesTotal)
		}
	})

	t.Run("sub-board carries its title and live counts", func(t *testing.T) {
		root := tuiTree(t)
		snap := mustScan(t, root)
		b := snap.Boards["."]
		if len(b.Subs) != 1 {
			t.Fatalf("subs = %+v, want one", b.Subs)
		}
		s := b.Subs[0]
		if s.Path != "backend" || s.Name != "backend/" || s.Title != "Backend" || s.Done != 1 || s.Total != 1 {
			t.Errorf("sub = %+v, want backend/ Backend 1/1", s)
		}
		if b.Done != 1 || b.Total != 3 {
			t.Errorf("root subtree = %d/%d, want 1/3", b.Done, b.Total)
		}
		if sub := snap.Boards["backend"]; sub.Parent != "." {
			t.Errorf("backend parent = %q, want .", sub.Parent)
		}
	})

	t.Run("status drift is flagged, file wins", func(t *testing.T) {
		root := tuiTree(t)
		rewriteBoard(t, root, func(c []byte) ([]byte, error) {
			return board.SetRowStatus(c, "T-02-beta-task.md", "done")
		})
		r, _ := scanRow(mustScan(t, root), ".", "T-02")
		if r.Status != "todo" || r.Drift != "board says done" {
			t.Errorf("row = %+v, want status todo drift %q", r, "board says done")
		}
	})

	t.Run("task without a row is appended with drift", func(t *testing.T) {
		root := tuiTree(t)
		rewriteBoard(t, root, func(c []byte) ([]byte, error) {
			return board.DropRow(c, "T-01-alpha-task.md")
		})
		b := mustScan(t, root).Boards["."]
		open := b.Sections[0]
		last := open.Rows[len(open.Rows)-1]
		if open.Name != "Open tasks" || last.ID != "T-01" || last.Drift != "no row" {
			t.Errorf("last row of %q = %+v, want T-01 with drift %q", open.Name, last, "no row")
		}
	})

	t.Run("row without a file is flagged", func(t *testing.T) {
		root := tuiTree(t)
		if err := os.Remove(filepath.Join(root, "T-02-beta-task.md")); err != nil {
			t.Fatal(err)
		}
		r, ok := scanRow(mustScan(t, root), ".", "T-02")
		if !ok {
			t.Fatal("T-02 row vanished with its file; the board row should remain")
		}
		if r.Drift != "no file" || r.Path != "" {
			t.Errorf("row = %+v, want drift %q and empty path", r, "no file")
		}
	})

	t.Run("archived tasks are invisible", func(t *testing.T) {
		root := tuiTree(t)
		mustDuty(t, root, "status", "T-01", "done")
		mustDuty(t, root, "archive")
		snap := mustScan(t, root)
		if _, ok := scanRow(snap, ".", "T-01"); ok {
			t.Error("archived T-01 still in snapshot")
		}
		b := snap.Boards["."]
		if b.Done != 0 || b.Total != 1 {
			t.Errorf("root subtree = %d/%d, want 0/1 (archive sweeps done tasks below too)", b.Done, b.Total)
		}
		if sub := b.Subs[0]; sub.Done != 0 || sub.Total != 0 {
			t.Errorf("backend counts = %d/%d, want 0/0", sub.Done, sub.Total)
		}
	})
}

// newTUIModel builds a model on the fixture tree with a fixed theme.
func newTUIModel(t *testing.T, root string) tui.Model {
	t.Helper()
	cfg := config.Config{Editor: "vi"}
	cfg.TUI.Theme = "dark"
	m, err := tui.New(root, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return nm.(tui.Model)
}

// press sends one key to the model. Board items on the fixture root:
// 0 backend/, 1 T-01, 2 T-02.
func press(t *testing.T, m tui.Model, k string) (tui.Model, tea.Cmd) {
	t.Helper()
	var msg tea.KeyMsg
	switch k {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	nm, cmd := m.Update(msg)
	return nm.(tui.Model), cmd
}

func TestModelTransitions(t *testing.T) {
	root := tuiTree(t)

	t.Run("cursor moves and clamps", func(t *testing.T) {
		m := newTUIModel(t, root)
		if m.BoardPath() != "." || m.Cursor() != 0 || m.DetailID() != "" {
			t.Fatalf("initial state: path=%q cursor=%d detail=%q", m.BoardPath(), m.Cursor(), m.DetailID())
		}
		for i, want := range []int{1, 2, 2} {
			m, _ = press(t, m, "j")
			if m.Cursor() != want {
				t.Fatalf("after %d j: cursor = %d, want %d", i+1, m.Cursor(), want)
			}
		}
		m, _ = press(t, m, "k")
		if m.Cursor() != 1 {
			t.Errorf("after k: cursor = %d, want 1", m.Cursor())
		}
	})

	t.Run("enter descends, esc climbs, cursor is remembered", func(t *testing.T) {
		m := newTUIModel(t, root)
		m, _ = press(t, m, "enter")
		if m.BoardPath() != "backend" || m.DetailID() != "" {
			t.Fatalf("after enter on sub-board: path=%q detail=%q", m.BoardPath(), m.DetailID())
		}
		m, _ = press(t, m, "esc")
		if m.BoardPath() != "." {
			t.Fatalf("after esc: path = %q, want .", m.BoardPath())
		}
		m, _ = press(t, m, "j")
		m, _ = press(t, m, "enter")
		m, _ = press(t, m, "esc")
		if m.Cursor() != 1 {
			t.Errorf("cursor not remembered: %d, want 1", m.Cursor())
		}
	})

	t.Run("esc on the root board is a no-op", func(t *testing.T) {
		m := newTUIModel(t, root)
		m, _ = press(t, m, "esc")
		if m.BoardPath() != "." {
			t.Errorf("path = %q, want .", m.BoardPath())
		}
	})

	t.Run("enter on a task opens detail, esc closes it", func(t *testing.T) {
		m := newTUIModel(t, root)
		m, _ = press(t, m, "j")
		m, _ = press(t, m, "enter")
		if m.DetailID() != "T-01" || m.BoardPath() != "." {
			t.Fatalf("detail=%q path=%q, want T-01 on .", m.DetailID(), m.BoardPath())
		}
		m, _ = press(t, m, "j")
		if m.DetailID() != "T-01" {
			t.Fatalf("j in detail closed it: detail=%q", m.DetailID())
		}
		m, _ = press(t, m, "esc")
		if m.DetailID() != "" || m.BoardPath() != "." {
			t.Errorf("after esc: detail=%q path=%q", m.DetailID(), m.BoardPath())
		}
	})

	t.Run("q quits from both views", func(t *testing.T) {
		m := newTUIModel(t, root)
		_, cmd := press(t, m, "q")
		if cmd == nil {
			t.Fatal("q returned no command")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("q command = %T, want tea.QuitMsg", cmd())
		}
		m, _ = press(t, m, "j")
		m, _ = press(t, m, "enter")
		_, cmd = press(t, m, "q")
		if cmd == nil {
			t.Fatal("q in detail returned no command")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("q in detail command = %T, want tea.QuitMsg", cmd())
		}
	})

	t.Run("e edits tasks, not sub-boards", func(t *testing.T) {
		m := newTUIModel(t, root)
		if _, cmd := press(t, m, "e"); cmd != nil {
			t.Error("e on a sub-board row returned a command")
		}
		m, _ = press(t, m, "j")
		if _, cmd := press(t, m, "e"); cmd == nil {
			t.Error("e on a task row returned no command")
		}
	})
}

func TestViewRendersHeadless(t *testing.T) {
	root := tuiTree(t)
	rewriteBoard(t, root, func(c []byte) ([]byte, error) {
		return board.SetRowStatus(c, "T-02-beta-task.md", "done")
	})
	m := newTUIModel(t, root)
	frame := m.View()
	if frame == "" {
		t.Fatal("board view is empty")
	}
	t.Logf("board view 100x30:\n%s", frame)

	m, _ = press(t, m, "j")
	m, _ = press(t, m, "enter")
	detail := m.View()
	if detail == "" {
		t.Fatal("detail view is empty")
	}
	t.Logf("detail view 100x30:\n%s", detail)

	nm, _ := m.Update(tea.WindowSizeMsg{Width: 38, Height: 10})
	if nm.(tui.Model).View() == "" {
		t.Fatal("narrow detail view is empty")
	}
}

// waitTick expects one watcher notification within a generous timeout.
func waitTick(t *testing.T, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(5 * time.Second):
		t.Fatal("watcher: no notification")
	}
}

// drainTicks swallows trailing notifications from the previous burst.
func drainTicks(c <-chan struct{}) {
	for {
		select {
		case <-c:
		case <-time.After(200 * time.Millisecond):
			return
		}
	}
}

func TestWatcherRefresh(t *testing.T) {
	root := tuiTree(t)
	w, err := tui.NewWatcher(root)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	mustDuty(t, root, "status", "T-02", "in-progress")
	waitTick(t, w.C)
	if r, _ := scanRow(mustScan(t, root), ".", "T-02"); r.Status != "in-progress" {
		t.Errorf("after external status: T-02 = %q, want in-progress", r.Status)
	}
	drainTicks(w.C)

	mustDuty(t, root, "board", "frontend", "--title", "Frontend")
	waitTick(t, w.C)
	drainTicks(w.C)

	mustDuty(t, filepath.Join(root, "frontend"), "create", "Delta task")
	waitTick(t, w.C)
	snap := mustScan(t, root)
	if _, ok := snap.Boards["frontend"]; !ok {
		t.Fatal("new sub-board missing from snapshot")
	}
	if r, ok := scanRow(snap, "frontend", "T-04"); !ok || r.Title != "Delta task" {
		t.Errorf("task created in new sub-board = %+v, want Delta task", r)
	}
	drainTicks(w.C)

	path := filepath.Join(root, "T-01-alpha-task.md")
	content := strings.Replace(readText(t, path), "## Gates\n", "## Gates\n\n- [x] saved\n", 1)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	waitTick(t, w.C)
	if r, _ := scanRow(mustScan(t, root), ".", "T-01"); r.GatesDone != 1 {
		t.Errorf("after editor-style save: gates = %d/%d, want 1/1", r.GatesDone, r.GatesTotal)
	}
}
