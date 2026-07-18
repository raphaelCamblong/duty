package tests

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/tui"
)

// archiveTree builds a fixture exercising every archive case: the root holds
// one open task (T-01) and one archived task (T-02); the "gone" track is
// archived-out (its only task T-03 was archived, subtree now empty); the
// "empty" track never held a task at all.
func archiveTree(t *testing.T) string {
	t.Helper()
	root := initDuty(t)
	mustDuty(t, root, "create", "task", "Root open task")
	mustDuty(t, root, "create", "task", "Root archived task")
	mustDuty(t, root, "create", "track", "gone", "--title", "Gone track")
	mustDuty(t, filepath.Join(root, "gone"), "create", "task", "Gone task")
	mustDuty(t, root, "create", "track", "empty", "--title", "Empty track")
	mustDuty(t, root, "status", "T-02", "done")
	mustDuty(t, root, "status", "T-03", "done")
	mustDuty(t, root, "archive")
	return root
}

// countingFS wraps an FS and tallies every ReadFile of a file inside an
// archive/ directory — the archived-content reads the cost-discipline rule
// forbids while the toggle is off.
type countingFS struct {
	fsys.FS
	archiveReads int
}

// ReadFile counts a read of an archived file's bytes, then delegates.
func (c *countingFS) ReadFile(name string) ([]byte, error) {
	if strings.Contains(filepath.ToSlash(name), "/"+names.ArchiveDir+"/") {
		c.archiveReads++
	}
	return c.FS.ReadFile(name)
}

func TestArchiveReadsOnlyWhenOn(t *testing.T) {
	root := archiveTree(t)

	off := &countingFS{FS: fsys.OS{}}
	if _, err := tui.Scan(off, root, false); err != nil {
		t.Fatalf("Scan(off) error = %v", err)
	}
	if off.archiveReads != 0 {
		t.Errorf("archive content reads while off = %d, want 0 (cost discipline)", off.archiveReads)
	}

	on := &countingFS{FS: fsys.OS{}}
	if _, err := tui.Scan(on, root, true); err != nil {
		t.Fatalf("Scan(on) error = %v", err)
	}
	if on.archiveReads != 2 {
		t.Errorf("archive content reads while on = %d, want 2 (T-02 and T-03)", on.archiveReads)
	}
}

func TestArchiveSnapshotCounts(t *testing.T) {
	root := archiveTree(t)

	off, err := tui.Scan(fsys.OS{}, root, false)
	if err != nil {
		t.Fatalf("Scan(off) error = %v", err)
	}
	rootBoard := off.Boards["."]
	if rootBoard.ArchivedCount != 1 {
		t.Errorf("root ArchivedCount = %d, want 1 (T-02)", rootBoard.ArchivedCount)
	}
	if rootBoard.Archived != nil {
		t.Errorf("root Archived rows = %v while off, want nil (no content read)", rootBoard.Archived)
	}
	gone, empty := subByName(t, rootBoard, "gone/"), subByName(t, rootBoard, "empty/")
	if gone.Total != 0 || gone.Archived != 1 {
		t.Errorf("gone track = %d open / %d archived, want 0 / 1 (archived-out)", gone.Total, gone.Archived)
	}
	if empty.Total != 0 || empty.Archived != 0 {
		t.Errorf("empty track = %d open / %d archived, want 0 / 0 (never-had)", empty.Total, empty.Archived)
	}

	on, err := tui.Scan(fsys.OS{}, root, true)
	if err != nil {
		t.Fatalf("Scan(on) error = %v", err)
	}
	rows := on.Boards["."].Archived
	if len(rows) != 1 || rows[0].ID != "T-02" || len(rows[0].Content) == 0 {
		t.Fatalf("root Archived rows = %+v, want one T-02 with content", rows)
	}
	if goneRows := on.Boards["gone"].Archived; len(goneRows) != 1 || goneRows[0].ID != "T-03" {
		t.Errorf("gone Archived rows = %+v, want one T-03", goneRows)
	}
}

// subByName finds a board's sub-track by its trailing-slash name.
func subByName(t *testing.T, b tui.Board, name string) tui.Sub {
	t.Helper()
	for _, s := range b.Subs {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("board has no track %q", name)
	return tui.Sub{}
}

func TestArchiveHidingRule(t *testing.T) {
	root := archiveTree(t)
	m := newTUIModelSize(t, root, 120, 35)

	off := m.View()
	if strings.Contains(off, "gone/") {
		t.Errorf("archived-out track shown while off:\n%s", off)
	}
	if !strings.Contains(off, "empty/") {
		t.Errorf("never-had track hidden while off (should stay as a container):\n%s", off)
	}
	if strings.Contains(off, "Archived (") {
		t.Errorf("archived section shown while off:\n%s", off)
	}
	if !strings.Contains(off, "T-01") {
		t.Errorf("open task missing while off:\n%s", off)
	}

	m = pressArchive(t, m)
	if !m.ShowArchive() {
		t.Fatal("a did not turn the archive view on")
	}
	on := m.View()
	if !strings.Contains(on, "gone/") {
		t.Errorf("archived-out track did not reappear while on:\n%s", on)
	}
	if !strings.Contains(on, "1 archived") {
		t.Errorf("reappeared track missing its archived count:\n%s", on)
	}
	if !strings.Contains(on, "Archived (1)") {
		t.Errorf("root archived section missing while on:\n%s", on)
	}
	if !strings.Contains(on, "T-02") {
		t.Errorf("archived row missing while on:\n%s", on)
	}

	m = pressArchive(t, m)
	if m.ShowArchive() {
		t.Error("a did not turn the archive view back off (session toggle)")
	}
	if strings.Contains(m.View(), "gone/") {
		t.Error("archived-out track still shown after toggling off")
	}

	m, _ = press(t, m, "?")
	if grid := m.View(); !strings.Contains(grid, "archive") {
		t.Errorf("? help grid missing the archive key:\n%s", grid)
	}
}

func TestArchivePreviewOpensReadOnly(t *testing.T) {
	root := archiveTree(t)
	m := newTUIModelSize(t, root, 120, 35)
	m = pressArchive(t, m)

	m = selectID(t, m, "T-02")
	m, _ = press(t, m, "enter")
	if m.DetailID() != "T-02" || !m.PreviewOpen() {
		t.Fatalf("enter on the archived row: detail=%q open=%v, want the T-02 preview", m.DetailID(), m.PreviewOpen())
	}
	if !strings.Contains(m.View(), "Goal") {
		t.Errorf("archived preview did not render the file body:\n%s", m.View())
	}

	m, _ = press(t, m, "esc")
	m = selectID(t, m, "T-02")
	if _, cmd := press(t, m, "e"); cmd != nil {
		t.Error("e on an archived row returned an edit command; archived tasks are read-only")
	}
}

// pressArchive toggles the archive view and applies the re-scan the toggle
// dispatches, so the returned model reflects the freshly-read archive.
func pressArchive(t *testing.T, m tui.Model) tui.Model {
	t.Helper()
	m, cmd := press(t, m, "a")
	return applyScan(t, m, cmd)
}

// applyScan runs a command the model returned and feeds its message back,
// unwrapping a batch — enough of the program loop to land a re-scan's snapshot.
func applyScan(t *testing.T, m tui.Model, cmd tea.Cmd) tui.Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			m = applyScan(t, m, c)
		}
		return m
	}
	nm, _ := m.Update(msg)
	return nm.(tui.Model)
}

// selectID moves the list selection onto the entry with the given id, failing
// if no row carries it.
func selectID(t *testing.T, m tui.Model, id string) tui.Model {
	t.Helper()
	for i := 0; i < 50; i++ {
		if m.SelectedID() == id {
			return m
		}
		m, _ = press(t, m, "j")
	}
	t.Fatalf("no selectable row with id %q", id)
	return m
}
