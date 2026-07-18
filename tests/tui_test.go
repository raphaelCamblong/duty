package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/tui"
	"github.com/raphaelCamblong/duty/internal/watch"
)

// tuiTree builds a fixture tree via the CLI: root tasks T-01 (in-progress)
// and T-02 (todo), and a "backend" sub-board holding T-03 (done).
func tuiTree(t *testing.T) string {
	t.Helper()
	root := initDuty(t)
	mustDuty(t, root, "create", "task", "Alpha task")
	mustDuty(t, root, "create", "task", "Beta task")
	mustDuty(t, root, "create", "track", "backend", "--title", "Backend")
	mustDuty(t, filepath.Join(root, "backend"), "create", "task", "Gamma task")
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
	snap, err := tui.Scan(fsys.OS{}, root, false)
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
	path := filepath.Join(dir, names.BoardFile)
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
		mustDuty(t, root, "move", "T-01", "--section", "Later")
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

const alphaGoal = "Ship the alpha milestone without regressions."

// fourStatusTree builds a fixture exercising all four statuses with a Goal on
// T-01: root T-01 (in-progress, goal filled) and T-02 (todo); a "backend"
// sub-board with T-03 (done) and T-04 (blocked).
func fourStatusTree(t *testing.T) string {
	t.Helper()
	root := initDuty(t)
	mustDuty(t, root, "create", "task", "Alpha task")
	mustDuty(t, root, "create", "task", "Beta task")
	mustDuty(t, root, "create", "track", "backend", "--title", "Backend")
	mustDuty(t, filepath.Join(root, "backend"), "create", "task", "Gamma task")
	mustDuty(t, filepath.Join(root, "backend"), "create", "task", "Delta task")
	mustDuty(t, root, "status", "T-01", "in-progress")
	mustDuty(t, root, "status", "T-03", "done")
	mustDuty(t, root, "status", "T-04", "blocked")
	writeGoal(t, filepath.Join(root, "T-01-alpha-task.md"), alphaGoal)
	return root
}

// writeGoal fills the empty "## Goal" section of a task file with text.
func writeGoal(t *testing.T, path, goal string) {
	t.Helper()
	content := strings.Replace(readText(t, path), "## Goal\n", "## Goal\n"+goal+"\n", 1)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write goal: %v", err)
	}
}

// sameCounts reports whether two status→count maps agree on every status,
// treating a missing key as zero.
func sameCounts(got, want map[string]int) bool {
	for _, st := range []string{"todo", "in-progress", "blocked", "done"} {
		if got[st] != want[st] {
			return false
		}
	}
	return true
}

func TestCountsRollUpInSnapshot(t *testing.T) {
	root := fourStatusTree(t)
	snap := mustScan(t, root)
	b := snap.Boards["."]
	wantRoot := map[string]int{"in-progress": 1, "todo": 1, "done": 1, "blocked": 1}
	if !sameCounts(b.Counts, wantRoot) {
		t.Errorf("root counts = %v, want %v", b.Counts, wantRoot)
	}
	sub := b.Subs[0]
	wantSub := map[string]int{"done": 1, "blocked": 1}
	if !sameCounts(sub.Counts, wantSub) {
		t.Errorf("backend counts = %v, want %v", sub.Counts, wantSub)
	}
}

func TestTrackBarCells(t *testing.T) {
	statuses := []string{"todo", "in-progress", "blocked", "done"}

	t.Run("proportional, every non-zero status visible, fixed width", func(t *testing.T) {
		cells := tui.BarCells(map[string]int{"done": 10, "todo": 2, "in-progress": 1, "blocked": 1}, 14)
		sum := 0
		for _, st := range statuses {
			if cells[st] < 1 {
				t.Errorf("status %q got %d cells, want >=1 (non-zero stays visible)", st, cells[st])
			}
			sum += cells[st]
		}
		if sum != 14 {
			t.Errorf("bar cells sum = %d, want 14 (fixed width)", sum)
		}
		if cells["done"] <= cells["todo"] {
			t.Errorf("dominant done=%d not wider than todo=%d (not proportional)", cells["done"], cells["todo"])
		}
	})

	t.Run("all five statuses each keep at least one cell", func(t *testing.T) {
		counts := map[string]int{"in-progress": 20, "todo": 1, "blocked": 1, "backlog": 1, "done": 1}
		cells := tui.BarCells(counts, 14)
		sum := 0
		for _, st := range []string{"in-progress", "todo", "blocked", "backlog", "done"} {
			if cells[st] < 1 {
				t.Errorf("status %q got %d cells, want >=1 (min-1-cell rule, 5 segments)", st, cells[st])
			}
			sum += cells[st]
		}
		if sum != 14 {
			t.Errorf("five-segment bar cells sum = %d, want 14 (fixed width)", sum)
		}
	})

	t.Run("width equal to the segment count gives each exactly one cell", func(t *testing.T) {
		cells := tui.BarCells(map[string]int{"in-progress": 9, "todo": 1, "blocked": 1, "backlog": 1, "done": 1}, 5)
		for _, st := range []string{"in-progress", "todo", "blocked", "backlog", "done"} {
			if cells[st] != 1 {
				t.Errorf("status %q got %d cells at width 5, want exactly 1", st, cells[st])
			}
		}
	})

	t.Run("single status fills the whole bar", func(t *testing.T) {
		if cells := tui.BarCells(map[string]int{"in-progress": 3}, 14); cells["in-progress"] != 14 {
			t.Errorf("single-status cells = %v, want in-progress: 14", cells)
		}
	})

	t.Run("empty subtree yields no bar", func(t *testing.T) {
		if cells := tui.BarCells(map[string]int{}, 14); cells != nil {
			t.Errorf("empty counts = %v, want nil", cells)
		}
	})
}

func TestTracksHeaderAndInlineBar(t *testing.T) {
	root := fourStatusTree(t) // root has a backend track (T-03 done, T-04 blocked)
	m := newTUIModelSize(t, root, 120, 35)
	frame := m.View().Content

	if !strings.Contains(frame, "Tracks") {
		t.Errorf("Tracks section header missing:\n%s", frame)
	}
	trackRow := strings.Split(frame, "\n")[lineWith(t, frame, "backend/", 38)]
	if !strings.Contains(trackRow, "█") {
		t.Errorf("track row missing the inline state bar: %q", trackRow)
	}

	if m.SelectedID() != "backend" {
		t.Fatalf("initial selection = %q, want backend (Tracks header skipped)", m.SelectedID())
	}
	m, _ = press(t, m, "k") // at the top: the header above must not be selectable
	if m.SelectedID() != "backend" {
		t.Errorf("k at the top: selection = %q, want backend (header not selectable)", m.SelectedID())
	}
	for _, want := range []string{"T-01", "T-02", "T-02"} {
		m, _ = press(t, m, "j")
		if m.SelectedID() != want {
			t.Fatalf("j lands on %q, want %q (section headers skipped)", m.SelectedID(), want)
		}
	}

	m = newTUIModelSize(t, root, 120, 35)
	var cmd tea.Cmd
	for _, k := range []string{"/", "b", "a", "c", "k"} {
		m, cmd = press(t, m, k)
		m = pump(t, m, cmd)
	}
	// Bubble Tea v2 always renders color, so the fuzzy filter underlines the
	// matched runes ("back") in place — splitting "backend/" with escapes. Strip
	// them to assert on the visible text, as the v1 non-tty tests did implicitly.
	filtered := ansi.Strip(m.View().Content)
	if strings.Contains(filtered, "Tracks") {
		t.Errorf("Tracks header still shown while filtering:\n%s", filtered)
	}
	if !strings.Contains(filtered, "backend/") {
		t.Errorf("filter dropped the matching track:\n%s", filtered)
	}
}

// twoTrackTree builds a root board with two sibling tracks whose title lengths
// and subtree counts differ: "api/" (short title, 2 tasks → 1-digit count) and
// "frontend/" (long title, 11 tasks → 2-digit count). The mismatch is what the
// right-aligned bar must survive — a left-inlined bar would start at a
// different x on each row.
func twoTrackTree(t *testing.T) string {
	t.Helper()
	root := initDuty(t)
	mustDuty(t, root, "create", "track", "api", "--title", "API")
	mustDuty(t, filepath.Join(root, "api"), "create", "task", "One api task")
	mustDuty(t, filepath.Join(root, "api"), "create", "task", "Two api task")
	mustDuty(t, root, "create", "track", "frontend", "--title", "The frontend web application")
	for i := 0; i < 11; i++ {
		mustDuty(t, filepath.Join(root, "frontend"), "create", "task", fmt.Sprintf("Frontend task %d", i))
	}
	return root
}

// barStartCol is the visual column of the first state-bar cell in a track row.
func barStartCol(t *testing.T, row string) int {
	t.Helper()
	plain := ansi.Strip(row)
	at := strings.Index(plain, "█")
	if at < 0 {
		t.Fatalf("track row has no state bar: %q", plain)
	}
	return ansi.StringWidth(plain[:at])
}

// contentEndCol is the visual column just past the row's last visible content,
// the panel padding and border trimmed — where the right-aligned count ends.
func contentEndCol(row string) int {
	return ansi.StringWidth(strings.TrimRight(ansi.Strip(row), " │"))
}

func TestTrackBarRightAligned(t *testing.T) {
	root := twoTrackTree(t)
	for _, sz := range []struct{ w, h int }{{120, 35}, {70, 20}} {
		t.Run(fmt.Sprintf("%dx%d", sz.w, sz.h), func(t *testing.T) {
			m := newTUIModelSize(t, root, sz.w, sz.h)
			frame := m.View().Content
			lines := strings.Split(frame, "\n")
			apiRow := lines[lineWith(t, frame, "api/", 40)]
			feRow := lines[lineWith(t, frame, "frontend/", 40)]

			apiBar, feBar := barStartCol(t, apiRow), barStartCol(t, feRow)
			if apiBar != feBar {
				t.Errorf("bar start columns differ (not aligned): api=%d frontend=%d\napi: %q\nfe:  %q",
					apiBar, feBar, ansi.Strip(apiRow), ansi.Strip(feRow))
			}
			if apiBar < sz.w/2 {
				t.Errorf("bar starts at col %d, left of the %d half: not right-aligned", apiBar, sz.w/2)
			}
			apiEnd, feEnd := contentEndCol(apiRow), contentEndCol(feRow)
			if apiEnd != feEnd {
				t.Errorf("count blocks not flush to a common right edge: api=%d frontend=%d", apiEnd, feEnd)
			}
			assertNoRagged(t, frame, sz.w)
			t.Logf("track rows %dx%d (bar flush right, aligned start col %d):\n%s\n%s",
				sz.w, sz.h, apiBar, ansi.Strip(apiRow), ansi.Strip(feRow))
		})
	}
}

func TestMasterDetailLayout(t *testing.T) {
	root := fourStatusTree(t)

	t.Run("opening a track with the preview open shows its summary card", func(t *testing.T) {
		m := newTUIModelSize(t, root, 120, 35)
		m, _ = press(t, m, "j")     // cursor 1: T-01
		m, _ = press(t, m, "enter") // open the preview so tracks summarize
		m, _ = press(t, m, "tab")   // focus the list
		m, _ = press(t, m, "k")     // cursor 0: the backend track
		m, _ = press(t, m, "enter") // open its summary card
		frame := m.View().Content
		for _, want := range []string{"1 blocked", "1 done", "2 tasks · 1 done", "Sections", "█"} {
			if !strings.Contains(frame, want) {
				t.Errorf("frame missing %q:\n%s", want, frame)
			}
		}
		t.Logf("board view 120x35 (track summary variant):\n%s", frame)
	})

	t.Run("opening a task previews its glamour-rendered body", func(t *testing.T) {
		m := newTUIModelSize(t, root, 120, 35)
		m, _ = press(t, m, "j")     // cursor 1: T-01
		m, _ = press(t, m, "enter") // open its preview
		frame := m.View().Content
		if !strings.Contains(frame, "Ship the alpha milestone") {
			t.Errorf("task body preview missing for T-01:\n%s", frame)
		}
		if !strings.Contains(frame, "in-progress") {
			t.Errorf("preview title missing the task status:\n%s", frame)
		}
		t.Logf("board view 120x35 (task preview variant):\n%s", frame)
	})

	t.Run("narrow terminal falls back to a single panel", func(t *testing.T) {
		m := newTUIModelSize(t, root, 70, 20)
		frame := m.View().Content
		if strings.Contains(frame, "Ship the alpha milestone") || strings.Contains(frame, "Sections") {
			t.Errorf("70x20 still renders a preview panel:\n%s", frame)
		}
		if !strings.Contains(frame, "T-01") {
			t.Errorf("70x20 list missing its rows:\n%s", frame)
		}
		t.Logf("board view 70x20 (single panel):\n%s", frame)

		m, _ = press(t, m, "j")
		m, _ = press(t, m, "enter")
		full := m.View().Content
		if !m.PreviewFocused() || !strings.Contains(full, "Ship the alpha milestone") {
			t.Fatalf("enter did not open the full-screen preview:\n%s", full)
		}
		if strings.Contains(full, "T-02") {
			t.Errorf("full-screen preview still shows the list:\n%s", full)
		}
		t.Logf("board view 70x20 (full-screen preview):\n%s", full)

		m, _ = press(t, m, "esc")
		if m.PreviewFocused() {
			t.Error("esc did not close the full-screen preview")
		}
	})
}

// perfTree builds a realistic fixture: three tracks and twenty tasks spread
// across the root and the tracks, for the startup-timing measurement.
func perfTree(t *testing.T) string {
	t.Helper()
	root := initDuty(t)
	tracks := []string{"backend", "frontend", "infra"}
	for _, tr := range tracks {
		mustDuty(t, root, "create", "track", tr, "--title", tr)
	}
	dirs := []string{
		root,
		filepath.Join(root, "backend"),
		filepath.Join(root, "frontend"),
		filepath.Join(root, "infra"),
	}
	for i := 0; i < 20; i++ {
		mustDuty(t, dirs[i%len(dirs)], "create", "task", fmt.Sprintf("Task %d does something useful", i))
	}
	return root
}

func TestStartupPerformance(t *testing.T) {
	root := perfTree(t)
	cfg := config.Config{Editor: "vi"}
	cfg.TUI.Theme = "dark"

	var best time.Duration
	for i := 0; i < 5; i++ {
		start := time.Now()
		m, err := tui.New(fsys.OS{}, root, cfg)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
		m = nm.(tui.Model)
		_ = m.View().Content
		if d := time.Since(start); best == 0 || d < best {
			best = d
		}
		if m.PreviewOpen() {
			t.Fatal("preview open at startup; browsing must be a full-width list")
		}
	}
	t.Logf("startup (New + WindowSize + View) on a 20-task/3-track fixture, best of 5: %s", best)
	if best > 100*time.Millisecond {
		t.Errorf("startup too slow: %s > 100ms (no terminal query, no glamour on this path)", best)
	}
}

func TestPreviewOnOpen(t *testing.T) {
	root := fourStatusTree(t)
	m := newTUIModelSize(t, root, 120, 35)

	browse := m.View().Content
	if m.PreviewOpen() {
		t.Fatal("a preview is open while browsing")
	}
	if strings.Contains(browse, "Ship the alpha milestone") {
		t.Errorf("browsing frame shows a task body:\n%s", browse)
	}
	for _, want := range []string{"backend/", "T-01", "T-02"} {
		if !strings.Contains(browse, want) {
			t.Errorf("browsing list missing %q:\n%s", want, browse)
		}
	}
	t.Logf("browsing 120x35 (full-width list, no preview):\n%s", browse)

	m, _ = press(t, m, "j")     // T-01
	m, _ = press(t, m, "enter") // open it
	if !m.PreviewOpen() || m.DetailID() != "T-01" {
		t.Fatalf("enter did not open the task: open=%v detail=%q", m.PreviewOpen(), m.DetailID())
	}
	open := m.View().Content
	if !strings.Contains(open, "Ship the alpha milestone") {
		t.Errorf("opened frame missing the task body:\n%s", open)
	}
	if !strings.Contains(open, "T-02") {
		t.Errorf("the split dropped the left list:\n%s", open)
	}
	t.Logf("task open 120x35 (split: list left, rendered task right):\n%s", open)

	m, _ = press(t, m, "esc") // close back to browsing
	if m.PreviewOpen() {
		t.Fatal("esc did not close the preview")
	}
	if strings.Contains(m.View().Content, "Ship the alpha milestone") {
		t.Errorf("closed frame still shows the task body:\n%s", m.View().Content)
	}

	m, _ = press(t, m, "k") // the backend track
	if m.SelectedID() != "backend" {
		t.Fatalf("selection = %q, want the backend track", m.SelectedID())
	}
	m, _ = press(t, m, "enter") // enter on a track descends, no panel
	if m.BoardPath() != "backend" || m.PreviewOpen() {
		t.Fatalf("enter on a track: path=%q open=%v, want descend into backend with no preview", m.BoardPath(), m.PreviewOpen())
	}
}

// pump executes a command a keypress returned and feeds any filter-match
// message back into the model, mirroring the program loop just enough for
// the list's asynchronous fuzzy filter.
func pump(t *testing.T, m tui.Model, cmd tea.Cmd) tui.Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			m = pump(t, m, c)
		}
		return m
	}
	if _, ok := msg.(list.FilterMatchesMsg); ok {
		nm, _ := m.Update(msg)
		return nm.(tui.Model)
	}
	return m
}

func TestPanelFocusAndFilter(t *testing.T) {
	root := fourStatusTree(t)

	t.Run("tab toggles focus only while a preview is open", func(t *testing.T) {
		m := newTUIModelSize(t, root, 120, 35)
		if m.PreviewFocused() || m.PreviewOpen() {
			t.Fatal("a preview is showing at start")
		}
		m, _ = press(t, m, "tab") // browsing: nothing to toggle
		if m.PreviewFocused() || m.PreviewOpen() {
			t.Fatal("tab opened a preview while browsing")
		}
		m, _ = press(t, m, "j")
		m, _ = press(t, m, "enter") // open T-01
		if !m.PreviewFocused() {
			t.Fatal("enter did not focus the preview")
		}
		m, _ = press(t, m, "tab") // focus the list, split stays open
		if m.PreviewFocused() || !m.PreviewOpen() {
			t.Fatalf("tab: focused=%v open=%v, want list focus with the split open", m.PreviewFocused(), m.PreviewOpen())
		}
		m, _ = press(t, m, "tab") // back to the preview
		if !m.PreviewFocused() {
			t.Fatal("tab did not return focus to the preview")
		}
		m, _ = press(t, m, "esc") // close to full-width browsing
		if m.PreviewFocused() || m.PreviewOpen() || m.BoardPath() != "." {
			t.Fatalf("esc: focused=%v open=%v path=%q, want browsing on .", m.PreviewFocused(), m.PreviewOpen(), m.BoardPath())
		}
	})

	t.Run("enter focuses the preview on a task, esc returns", func(t *testing.T) {
		m := newTUIModelSize(t, root, 120, 35)
		m, _ = press(t, m, "j")
		m, _ = press(t, m, "enter")
		if !m.PreviewFocused() || m.DetailID() != "T-01" {
			t.Fatalf("enter: focused=%v detail=%q, want preview on T-01", m.PreviewFocused(), m.DetailID())
		}
		m, _ = press(t, m, "esc")
		if m.PreviewFocused() || m.Cursor() != 1 {
			t.Errorf("esc: focused=%v cursor=%d, want list focus on 1", m.PreviewFocused(), m.Cursor())
		}
	})

	t.Run("filter narrows fuzzily and esc clears it", func(t *testing.T) {
		m := newTUIModelSize(t, root, 120, 35)
		var cmd tea.Cmd
		for _, k := range []string{"/", "b", "e", "t", "a"} {
			m, cmd = press(t, m, k)
			m = pump(t, m, cmd)
		}
		m, _ = press(t, m, "enter")
		if m.SelectedID() != "T-02" || m.Cursor() != 0 {
			t.Fatalf("filtered selection = %q at %d, want T-02 at 0", m.SelectedID(), m.Cursor())
		}
		if frame := m.View().Content; strings.Contains(frame, "Alpha task") {
			t.Errorf("filtered list still shows T-01:\n%s", frame)
		}
		m, _ = press(t, m, "esc")
		if frame := m.View().Content; !strings.Contains(frame, "Alpha task") {
			t.Errorf("esc did not clear the filter:\n%s", frame)
		}
		if m.BoardPath() != "." {
			t.Errorf("esc while filtered climbed to %q", m.BoardPath())
		}
	})

	t.Run("slash pulls focus back to the list", func(t *testing.T) {
		m := newTUIModelSize(t, root, 120, 35)
		m, _ = press(t, m, "j")
		m, _ = press(t, m, "enter") // open the preview, focus it
		m, _ = press(t, m, "/")
		if m.PreviewFocused() {
			t.Error("/ left the preview focused")
		}
	})

	t.Run("descending resets filter and remembers selection per track", func(t *testing.T) {
		m := newTUIModelSize(t, root, 120, 35)
		m, _ = press(t, m, "j") // T-01
		m, _ = press(t, m, "k") // back to backend/
		m, _ = press(t, m, "enter")
		// Backend's rows show status-sorted by default (T-34): blocked T-04
		// above done T-03, so the descent lands on T-04.
		if m.BoardPath() != "backend" || m.SelectedID() != "T-04" {
			t.Fatalf("descend: path=%q selected=%q, want backend / T-04", m.BoardPath(), m.SelectedID())
		}
		m, _ = press(t, m, "j") // T-03
		m, _ = press(t, m, "esc")
		if m.BoardPath() != "." || m.SelectedID() != "backend" {
			t.Fatalf("climb: path=%q selected=%q, want . / backend", m.BoardPath(), m.SelectedID())
		}
		m, _ = press(t, m, "enter")
		if m.SelectedID() != "T-03" {
			t.Errorf("re-descend selection = %q, want the remembered T-03", m.SelectedID())
		}
	})
}

// newTUIModel builds a model on the fixture tree, sized 100×30.
func newTUIModel(t *testing.T, root string) tui.Model {
	t.Helper()
	return newTUIModelSize(t, root, 100, 30)
}

// newTUIModelSize builds a model on the fixture tree at a given terminal size.
func newTUIModelSize(t *testing.T, root string, w, h int) tui.Model {
	t.Helper()
	cfg := config.Config{Editor: "vi"}
	cfg.TUI.Theme = "dark"
	m, err := tui.New(fsys.OS{}, root, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return nm.(tui.Model)
}

// clickAt sends one left button press at screen cell (x, y).
func clickAt(m tui.Model, x, y int) tui.Model {
	msg := tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}
	nm, _ := m.Update(msg)
	return nm.(tui.Model)
}

// wheelAt sends one wheel event at screen cell (x, y).
func wheelAt(m tui.Model, down bool, x, y int) tui.Model {
	b := tea.MouseWheelUp
	if down {
		b = tea.MouseWheelDown
	}
	nm, _ := m.Update(tea.MouseWheelMsg{X: x, Y: y, Button: b})
	return nm.(tui.Model)
}

// render draws one frame and gives the BubbleZone worker time to index the
// hit-zones the frame registered, so the next mouse event lands.
func render(t *testing.T, m tui.Model) string {
	t.Helper()
	frame := m.View().Content
	time.Sleep(100 * time.Millisecond)
	return frame
}

// lineWith returns the screen row of the first frame line showing s within
// the left panel (its first maxX columns), so right-panel echoes don't match.
func lineWith(t *testing.T, frame, s string, maxX int) int {
	t.Helper()
	for i, l := range strings.Split(frame, "\n") {
		plain := ansi.Strip(l)
		if at := strings.Index(plain, s); at >= 0 && ansi.StringWidth(plain[:at]) < maxX {
			return i
		}
	}
	t.Fatalf("frame has no line containing %q before column %d:\n%s", s, maxX, frame)
	return -1
}

// press sends one key to the model. Board items on the fixture root:
// 0 backend/, 1 T-01, 2 T-02.
func press(t *testing.T, m tui.Model, k string) (tui.Model, tea.Cmd) {
	t.Helper()
	var msg tea.KeyMsg
	switch k {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEsc}
	case "tab":
		msg = tea.KeyPressMsg{Code: tea.KeyTab}
	default:
		msg = tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
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

func TestMouseTransitions(t *testing.T) {
	root := fourStatusTree(t)

	t.Run("click selects the row under the pointer", func(t *testing.T) {
		m := newTUIModel(t, root)
		frame := render(t, m)
		m = clickAt(m, 6, lineWith(t, frame, "T-01", 38))
		if m.Cursor() != 1 || m.DetailID() != "" {
			t.Fatalf("click T-01: cursor=%d detail=%q, want 1 / no detail", m.Cursor(), m.DetailID())
		}
		frame = render(t, m)
		m = clickAt(m, 6, lineWith(t, frame, "T-02", 38))
		if m.Cursor() != 2 {
			t.Errorf("click T-02: cursor=%d, want 2", m.Cursor())
		}
	})

	t.Run("double-click a task focuses its preview", func(t *testing.T) {
		m := newTUIModel(t, root)
		y := lineWith(t, render(t, m), "T-01", 38)
		m = clickAt(m, 6, y)
		m = clickAt(m, 6, y)
		if m.DetailID() != "T-01" || m.BoardPath() != "." {
			t.Fatalf("double-click T-01: detail=%q path=%q, want T-01 on .", m.DetailID(), m.BoardPath())
		}
	})

	t.Run("double-click a track descends", func(t *testing.T) {
		m := newTUIModel(t, root)
		y := lineWith(t, render(t, m), "backend/", 38)
		m = clickAt(m, 6, y)
		if m.Cursor() != 0 {
			t.Fatalf("click backend: cursor=%d, want 0", m.Cursor())
		}
		m = clickAt(m, 6, y)
		if m.BoardPath() != "backend" {
			t.Errorf("double-click backend: path=%q, want backend", m.BoardPath())
		}
	})

	t.Run("click on the right panel focuses the preview", func(t *testing.T) {
		m := newTUIModel(t, root) // 100 wide: left panel ends at column 38
		m, _ = press(t, m, "j")
		m, _ = press(t, m, "enter") // open the split
		m, _ = press(t, m, "tab")   // hand focus to the list
		render(t, m)
		m = clickAt(m, 80, 10)
		if !m.PreviewFocused() {
			t.Error("preview click did not focus it")
		}
	})

	t.Run("a click on the header changes nothing", func(t *testing.T) {
		m := newTUIModel(t, root)
		render(t, m)
		m = clickAt(m, 6, 1)
		if m.Cursor() != 0 || m.DetailID() != "" || m.PreviewFocused() {
			t.Errorf("header click: cursor=%d detail=%q, want 0 / none", m.Cursor(), m.DetailID())
		}
	})

	t.Run("wheel over the preview springs its scroll and clamps", func(t *testing.T) {
		m := newTUIModelSize(t, root, 120, 16) // shallow preview: the body overflows
		m, _ = press(t, m, "j")                // select T-01
		m, _ = press(t, m, "enter")            // open its preview
		render(t, m)
		if m.ScrollTarget() != 0 {
			t.Fatalf("initial scroll target = %d, want 0", m.ScrollTarget())
		}
		m = wheelAt(m, true, 100, 8)
		if m.ScrollTarget() != 3 {
			t.Fatalf("one wheel down: target = %d, want 3", m.ScrollTarget())
		}
		m = wheelAt(m, false, 100, 8)
		m = wheelAt(m, false, 100, 8)
		if m.ScrollTarget() != 0 {
			t.Errorf("two wheel up: target = %d, want 0 (clamped)", m.ScrollTarget())
		}
	})

	t.Run("wheel over the list moves the selection", func(t *testing.T) {
		m := newTUIModel(t, root)
		render(t, m)
		m = wheelAt(m, true, 10, 8)
		if m.Cursor() != 1 {
			t.Fatalf("wheel down: cursor = %d, want 1", m.Cursor())
		}
		m = wheelAt(m, false, 10, 8)
		if m.Cursor() != 0 {
			t.Errorf("wheel up: cursor = %d, want 0", m.Cursor())
		}
	})
}

func TestHeaderBarAndHelpFooter(t *testing.T) {
	root := tuiTree(t)
	m := newTUIModel(t, root)

	frame := m.View().Content
	if !strings.Contains(frame, "█") {
		t.Errorf("header status-distribution bar missing from view:\n%s", frame)
	}
	if !strings.Contains(frame, "quit") || !strings.Contains(frame, " • ") {
		t.Errorf("short help hint footer missing from view:\n%s", frame)
	}
	if m.HelpExpanded() {
		t.Fatal("help should start collapsed")
	}

	m, _ = press(t, m, "?")
	if !m.HelpExpanded() {
		t.Fatal("? did not expand the help footer")
	}
	if full := m.View().Content; strings.Contains(full, " • ") {
		t.Errorf("help still showing the short bar after ?:\n%s", full)
	}

	m, _ = press(t, m, "?")
	if m.HelpExpanded() {
		t.Error("second ? did not collapse the help footer")
	}
	t.Logf("board view with header bar + help footer 100x30:\n%s", frame)
}

func TestViewRendersHeadless(t *testing.T) {
	root := tuiTree(t)
	rewriteBoard(t, root, func(c []byte) ([]byte, error) {
		return board.SetRowStatus(c, "T-02-beta-task.md", "done")
	})
	m := newTUIModel(t, root)
	frame := m.View().Content
	if frame == "" {
		t.Fatal("board view is empty")
	}
	t.Logf("board view 100x30:\n%s", frame)

	m, _ = press(t, m, "j")
	m, _ = press(t, m, "enter")
	detail := m.View().Content
	if detail == "" {
		t.Fatal("detail view is empty")
	}
	t.Logf("detail view 100x30:\n%s", detail)

	nm, _ := m.Update(tea.WindowSizeMsg{Width: 38, Height: 10})
	if nm.(tui.Model).View().Content == "" {
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
	w, err := watch.NewWatcher(fsys.OS{}, root)
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

	mustDuty(t, root, "create", "track", "frontend", "--title", "Frontend")
	waitTick(t, w.C)
	drainTicks(w.C)

	mustDuty(t, filepath.Join(root, "frontend"), "create", "task", "Delta task")
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

func TestManualRefresh(t *testing.T) {
	root := tuiTree(t)
	m := newTUIModel(t, root)
	if strings.Contains(m.View().Content, "T-04") {
		t.Fatalf("fixture already carries T-04:\n%s", m.View().Content)
	}

	mustDuty(t, root, "create", "task", "Epsilon task")

	_, cmd := press(t, m, "r")
	if cmd == nil {
		t.Fatal("r returned no re-scan command")
	}
	nm, _ := m.Update(cmd())
	m = nm.(tui.Model)

	frame := m.View().Content
	if !strings.Contains(frame, "T-04") || !strings.Contains(frame, "Epsilon task") {
		t.Errorf("r did not re-scan the new task into the list:\n%s", frame)
	}
}

func TestBreadcrumbClickNavigates(t *testing.T) {
	root := fourStatusTree(t)
	m := newTUIModel(t, root)
	m, _ = press(t, m, "enter") // descend into the backend track (cursor 0)
	if m.BoardPath() != "backend" {
		t.Fatalf("did not descend: path=%q", m.BoardPath())
	}

	frame := render(t, m)
	y := lineWith(t, frame, "Board", 100) // the model is 100 columns wide
	m = clickAt(m, 4, y)                  // "Board" is the root crumb, columns 2..6
	if m.BoardPath() != "." {
		t.Errorf("clicking the root breadcrumb did not jump to it: path=%q", m.BoardPath())
	}
}

func TestEmptyStates(t *testing.T) {
	t.Run("fresh tree names itself", func(t *testing.T) {
		root := initDuty(t)
		frame := newTUIModelSize(t, root, 100, 30).View().Content
		if !strings.Contains(frame, "empty tree") || !strings.Contains(frame, `duty create task`) {
			t.Errorf("fresh-tree hint missing:\n%s", frame)
		}
		t.Logf("fresh tree 100x30:\n%s", frame)
	})

	t.Run("empty track nudges toward create", func(t *testing.T) {
		root := initDuty(t)
		mustDuty(t, root, "create", "track", "backend", "--title", "Backend")
		m := newTUIModelSize(t, root, 100, 30)
		m, _ = press(t, m, "enter") // descend into the empty track
		if m.BoardPath() != "backend" {
			t.Fatalf("did not descend into the empty track: path=%q", m.BoardPath())
		}
		frame := m.View().Content
		if !strings.Contains(frame, "no tasks yet") || !strings.Contains(frame, `duty create task`) {
			t.Errorf("empty-track hint missing:\n%s", frame)
		}
		t.Logf("empty track 100x30:\n%s", frame)
	})

	t.Run("filter with no matches shows the styled no-items state", func(t *testing.T) {
		root := fourStatusTree(t)
		m := newTUIModelSize(t, root, 100, 30)
		var cmd tea.Cmd
		for _, k := range []string{"/", "z", "z", "z", "z"} {
			m, cmd = press(t, m, k)
			m = pump(t, m, cmd)
		}
		frame := m.View().Content
		if !strings.Contains(frame, "No matches") {
			t.Errorf("empty-filter no-items hint missing:\n%s", frame)
		}
		if strings.Contains(frame, "Alpha task") {
			t.Errorf("no-match filter still shows rows:\n%s", frame)
		}
		t.Logf("filter with no matches 100x30:\n%s", frame)
	})
}

// auditTree stresses the layout: tasks in every status, a blocked-by link, a
// filled goal, gate progress, and a titled track.
func auditTree(t *testing.T) string {
	t.Helper()
	root := initDuty(t)
	mustDuty(t, root, "create", "task", "Alpha task")
	mustDuty(t, root, "create", "task", "Beta task", "--blocked-by", "T-01")
	mustDuty(t, root, "create", "track", "backend", "--title", "Backend services")
	mustDuty(t, filepath.Join(root, "backend"), "create", "task", "Gamma task")
	mustDuty(t, filepath.Join(root, "backend"), "create", "task", "Delta task")
	mustDuty(t, root, "status", "T-01", "in-progress")
	mustDuty(t, root, "status", "T-03", "done")
	mustDuty(t, root, "status", "T-04", "blocked")
	writeGoal(t, filepath.Join(root, "T-01-alpha-task.md"), alphaGoal)
	return root
}

// assertNoRagged fails when any frame line is wider than the terminal.
func assertNoRagged(t *testing.T, frame string, w int) {
	t.Helper()
	for i, line := range strings.Split(frame, "\n") {
		if got := lipgloss.Width(line); got > w {
			t.Errorf("line %d width %d exceeds terminal width %d: %q", i, got, w, line)
		}
	}
}

func TestFrameAudit(t *testing.T) {
	root := auditTree(t)
	sizes := []struct{ w, h int }{{120, 35}, {100, 28}, {80, 24}, {70, 20}, {60, 16}}
	for _, sz := range sizes {
		name := fmt.Sprintf("%dx%d", sz.w, sz.h)
		t.Run(name, func(t *testing.T) {
			m := newTUIModelSize(t, root, sz.w, sz.h)
			browse := m.View().Content
			assertNoRagged(t, browse, sz.w)
			t.Logf("browse %s:\n%s", name, browse)

			m, _ = press(t, m, "j")     // T-01
			m, _ = press(t, m, "j")     // T-02 (todo, blocked-by T-01)
			m, _ = press(t, m, "enter") // open its preview
			open := m.View().Content
			assertNoRagged(t, open, sz.w)
			if !strings.Contains(open, "blocked-by T-01") {
				t.Errorf("preview header missing blocked-by at %s:\n%s", name, open)
			}
			t.Logf("preview %s:\n%s", name, open)
		})
	}
}

// scrambledStatusTree builds a flat board whose six tasks carry all four
// statuses out of order in the board file, for the display-sort tests: board
// order T-01..T-06 with statuses done, todo, in-progress, blocked, todo,
// in-progress.
func scrambledStatusTree(t *testing.T) string {
	t.Helper()
	root := initDuty(t)
	for _, title := range []string{"One", "Two", "Three", "Four", "Five", "Six"} {
		mustDuty(t, root, "create", "task", title)
	}
	mustDuty(t, root, "status", "T-01", "done")
	mustDuty(t, root, "status", "T-03", "in-progress")
	mustDuty(t, root, "status", "T-04", "blocked")
	mustDuty(t, root, "status", "T-06", "in-progress")
	return root
}

// fiveStatusTree builds a flat board whose five tasks carry all five statuses,
// board order T-01..T-05 deliberately reversed from the display sort: done,
// backlog, blocked, todo, in-progress. Sorting must reorder them to
// in-progress → todo → blocked → backlog → done (T-05..T-01).
func fiveStatusTree(t *testing.T) string {
	t.Helper()
	root := initDuty(t)
	for _, title := range []string{"One", "Two", "Three", "Four", "Five"} {
		mustDuty(t, root, "create", "task", title)
	}
	mustDuty(t, root, "status", "T-01", "done")
	mustDuty(t, root, "status", "T-02", "backlog")
	mustDuty(t, root, "status", "T-03", "blocked")
	mustDuty(t, root, "status", "T-05", "in-progress")
	return root
}

func TestStatusSortPlacesBacklogBeforeDone(t *testing.T) {
	root := fiveStatusTree(t)
	m := newTUIModelSize(t, root, 120, 35)

	// Default ON: in-progress → todo → blocked → backlog → done.
	want := []string{"T-05", "T-04", "T-03", "T-02", "T-01"}
	if got := rowOrder(t, m); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("status-sorted rows = %v, want %v (backlog between blocked and done)", got, want)
	}
}

// rowOrder walks the left panel top-to-bottom, returning each selectable id in
// display order (section headers skipped). It leaves the caller's model
// untouched (the model is a value).
func rowOrder(t *testing.T, m tui.Model) []string {
	t.Helper()
	for i := 0; i < 100; i++ {
		m, _ = press(t, m, "k")
	}
	ids := []string{m.SelectedID()}
	for i := 0; i < 100; i++ {
		nm, _ := press(t, m, "j")
		if nm.SelectedID() == m.SelectedID() {
			break
		}
		m = nm
		ids = append(ids, m.SelectedID())
	}
	return ids
}

func TestStatusSortedRows(t *testing.T) {
	root := scrambledStatusTree(t)
	m := newTUIModelSize(t, root, 120, 35)

	// Default ON: in-progress → todo → blocked → done, board order the stable
	// tiebreak within each group (T-03 before T-06, T-02 before T-05).
	want := []string{"T-03", "T-06", "T-02", "T-05", "T-04", "T-01"}
	if got := rowOrder(t, m); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("status-sorted rows = %v, want %v", got, want)
	}

	// Frame check: the in-progress head renders above the done tail.
	frame := m.View().Content
	if lineWith(t, frame, "T-03", 60) >= lineWith(t, frame, "T-01", 60) {
		t.Errorf("T-03 (in-progress) not rendered above T-01 (done):\n%s", frame)
	}
}

func TestStatusSortToggle(t *testing.T) {
	root := scrambledStatusTree(t)
	m := newTUIModelSize(t, root, 120, 35)

	sorted := []string{"T-03", "T-06", "T-02", "T-05", "T-04", "T-01"}
	boardOrder := []string{"T-01", "T-02", "T-03", "T-04", "T-05", "T-06"}

	if got := rowOrder(t, m); strings.Join(got, " ") != strings.Join(sorted, " ") {
		t.Fatalf("default rows = %v, want status-sorted %v", got, sorted)
	}
	m, cmd := press(t, m, "s") // toggle to raw board order
	m = pump(t, m, cmd)
	if got := rowOrder(t, m); strings.Join(got, " ") != strings.Join(boardOrder, " ") {
		t.Fatalf("after s: rows = %v, want board order %v", got, boardOrder)
	}
	m, cmd = press(t, m, "s") // toggle back to the status sort
	m = pump(t, m, cmd)
	if got := rowOrder(t, m); strings.Join(got, " ") != strings.Join(sorted, " ") {
		t.Errorf("after s again: rows = %v, want status-sorted %v", got, sorted)
	}
}

func TestStatusSortFilterInterplay(t *testing.T) {
	root := scrambledStatusTree(t)
	m := newTUIModelSize(t, root, 120, 35)

	var cmd tea.Cmd
	for _, k := range []string{"/", "o", "n", "e"} { // fuzzy-matches "One" = T-01
		m, cmd = press(t, m, k)
		m = pump(t, m, cmd)
	}
	m, _ = press(t, m, "enter")
	if m.SelectedID() != "T-01" {
		t.Fatalf("filter 'one' selected %q, want T-01 (fuzzy rank, not the status sort)", m.SelectedID())
	}
	m, _ = press(t, m, "esc") // clear the filter
	want := []string{"T-03", "T-06", "T-02", "T-05", "T-04", "T-01"}
	if got := rowOrder(t, m); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("clearing the filter did not restore the status sort: rows = %v, want %v", got, want)
	}
}
