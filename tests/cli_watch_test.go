package tests

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/raphaelCamblong/duty/internal/app"
	"github.com/raphaelCamblong/duty/internal/cli"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/watch"
)

// TestWatchDiff pins the pure snapshot diff: one Event per changed field,
// every event kind, and a deterministic order (by id, then a fixed field
// order).
func TestWatchDiff(t *testing.T) {
	tests := []struct {
		name          string
		before, after map[string]app.TaskState
		want          []app.Event
	}{
		{
			name:   "no change yields nothing",
			before: map[string]app.TaskState{"T-01": {Status: "todo", Board: "."}},
			after:  map[string]app.TaskState{"T-01": {Status: "todo", Board: "."}},
			want:   nil,
		},
		{
			name:   "status change",
			before: map[string]app.TaskState{"T-01": {Status: "todo", Board: "."}},
			after:  map[string]app.TaskState{"T-01": {Status: "done", Board: "."}},
			want:   []app.Event{{Kind: app.EventStatus, ID: "T-01", Field: "status", Old: "todo", New: "done"}},
		},
		{
			name:   "claim released",
			before: map[string]app.TaskState{"T-01": {Status: "in-progress", ClaimedBy: "a", Board: "."}},
			after:  map[string]app.TaskState{"T-01": {Status: "in-progress", Board: "."}},
			want:   []app.Event{{Kind: app.EventClaimedBy, ID: "T-01", Field: "claimed-by", Old: "a", New: ""}},
		},
		{
			name:   "created",
			before: map[string]app.TaskState{},
			after:  map[string]app.TaskState{"T-05": {Status: "todo", Board: "backend"}},
			want:   []app.Event{{Kind: app.EventCreated, ID: "T-05", Field: "status", New: "todo"}},
		},
		{
			name:   "deleted",
			before: map[string]app.TaskState{"T-05": {Status: "done", Board: "."}},
			after:  map[string]app.TaskState{},
			want:   []app.Event{{Kind: app.EventDeleted, ID: "T-05", Field: "status", Old: "done"}},
		},
		{
			name:   "moved",
			before: map[string]app.TaskState{"T-01": {Status: "todo", Board: "."}},
			after:  map[string]app.TaskState{"T-01": {Status: "todo", Board: "backend"}},
			want:   []app.Event{{Kind: app.EventMoved, ID: "T-01", Field: "board", Old: ".", New: "backend"}},
		},
		{
			name:   "gates progress",
			before: map[string]app.TaskState{"T-01": {Status: "in-progress", Board: ".", GatesDone: 0, GatesTotal: 2}},
			after:  map[string]app.TaskState{"T-01": {Status: "in-progress", Board: ".", GatesDone: 1, GatesTotal: 2}},
			want:   []app.Event{{Kind: app.EventGates, ID: "T-01", Field: "gates", Old: "0/2", New: "1/2"}},
		},
		{
			name:   "claiming records status then claimed-by in fixed order",
			before: map[string]app.TaskState{"T-02": {Status: "todo", Board: "."}},
			after:  map[string]app.TaskState{"T-02": {Status: "in-progress", ClaimedBy: "sonnet-3", Board: "."}},
			want: []app.Event{
				{Kind: app.EventStatus, ID: "T-02", Field: "status", Old: "todo", New: "in-progress"},
				{Kind: app.EventClaimedBy, ID: "T-02", Field: "claimed-by", New: "sonnet-3"},
			},
		},
		{
			name: "changes are ordered by id",
			before: map[string]app.TaskState{
				"T-02": {Status: "todo", Board: "."},
				"T-10": {Status: "todo", Board: "."},
			},
			after: map[string]app.TaskState{
				"T-02": {Status: "done", Board: "."},
				"T-10": {Status: "done", Board: "."},
			},
			want: []app.Event{
				{Kind: app.EventStatus, ID: "T-02", Field: "status", Old: "todo", New: "done"},
				{Kind: app.EventStatus, ID: "T-10", Field: "status", Old: "todo", New: "done"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.Diff(tt.before, tt.after)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Diff() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestWatchSnapshot reads a real tree into the state duty watch diffs: every
// open task keyed by id, its board path relative to the scanned board, and its
// gate counts.
func TestWatchSnapshot(t *testing.T) {
	root := tuiTree(t)
	mustDuty(t, root, "gates", "add", "T-01", "build passes")

	snap, err := app.New(fsys.OS{}).Snapshot(root, "")
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if got, want := len(snap), 3; got != want {
		t.Fatalf("snapshot has %d tasks, want %d: %+v", got, want, snap)
	}
	if s := snap["T-01"]; s.Status != "in-progress" || s.GatesTotal != 1 || s.Board != "." {
		t.Errorf("T-01 = %+v, want in-progress, 0/1 gates, board .", s)
	}
	if s := snap["T-03"]; s.Status != "done" || s.Board != "backend" {
		t.Errorf("T-03 = %+v, want done, board backend", s)
	}
}

// TestWatchStreamEvents drives CLI mutations while the shared watcher runs and
// asserts the exact events each burst diffs — every kind, mirroring the TUI's
// TestWatcherRefresh technique (synchronous watcher, mutate, wait for a tick,
// re-scan, compare).
func TestWatchStreamEvents(t *testing.T) {
	root := tuiTree(t)
	mustDuty(t, root, "gates", "add", "T-01", "build passes")

	a := app.New(fsys.OS{})
	w, err := watch.NewWatcher(fsys.OS{}, root)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()
	prev := mustSnapshot(t, a, root)

	mustDuty(t, root, "status", "T-02", "blocked")
	prev = wantBurst(t, a, root, w, prev,
		app.Event{Kind: app.EventStatus, ID: "T-02", Field: "status", Old: "todo", New: "blocked"})

	mustDuty(t, root, "status", "T-02", "in-progress", "--as", "sonnet-3")
	prev = wantBurst(t, a, root, w, prev,
		app.Event{Kind: app.EventStatus, ID: "T-02", Field: "status", Old: "blocked", New: "in-progress"},
		app.Event{Kind: app.EventClaimedBy, ID: "T-02", Field: "claimed-by", New: "sonnet-3"})

	mustDuty(t, root, "create", "task", "Delta task")
	prev = wantBurst(t, a, root, w, prev,
		app.Event{Kind: app.EventCreated, ID: "T-04", Field: "status", New: "todo"})

	mustDuty(t, root, "move", "T-04", "--track", "backend")
	prev = wantBurst(t, a, root, w, prev,
		app.Event{Kind: app.EventMoved, ID: "T-04", Field: "board", Old: ".", New: "backend"})

	mustDuty(t, root, "gates", "check", "T-01", "1")
	prev = wantBurst(t, a, root, w, prev,
		app.Event{Kind: app.EventGates, ID: "T-01", Field: "gates", Old: "0/1", New: "1/1"})

	mustDuty(t, root, "delete", "task", "T-04", "--force")
	wantBurst(t, a, root, w, prev,
		app.Event{Kind: app.EventDeleted, ID: "T-04", Field: "status", Old: "todo"})
}

// TestWatchCommandExit runs the real watch command: one mutation emits one
// correctly formatted TSV line, and the tree disappearing exits non-zero with
// one lowercase stderr line.
func TestWatchCommandExit(t *testing.T) {
	root := tuiTree(t)
	t.Chdir(root)

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	lines := streamLines(pr)
	var errBuf bytes.Buffer
	code := make(chan int, 1)
	go func() {
		code <- cli.Run([]string{"watch", "--agent"}, strings.NewReader(""), pw, &errBuf, "test")
		_ = pw.Close()
	}()

	time.Sleep(300 * time.Millisecond) // let the watcher establish before the first mutation
	mustDuty(t, root, "status", "T-02", "blocked")
	got := nextWatchLine(t, lines)
	want := []string{app.EventStatus, "T-02", "status", "todo", "blocked"}
	if !slices.Equal(got, want) {
		t.Errorf("watch line = %v, want %v", got, want)
	}

	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-code:
		if c == 0 {
			t.Errorf("watch exit = %d, want non-zero after the tree disappears", c)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("watch did not exit after the tree disappeared")
	}
	line := strings.TrimRight(errBuf.String(), "\n")
	if line == "" || strings.Contains(line, "\n") {
		t.Errorf("stderr = %q, want exactly one line", errBuf.String())
	}
	if r := rune(line[0]); r >= 'A' && r <= 'Z' || strings.HasSuffix(line, ".") {
		t.Errorf("stderr = %q, want a lowercase line with no trailing period", line)
	}
}

// mustSnapshot snapshots root's whole tree, failing the test on error.
func mustSnapshot(t *testing.T, a app.App, root string) map[string]app.TaskState {
	t.Helper()
	snap, err := a.Snapshot(root, "")
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	return snap
}

// wantBurst waits for one watcher tick, re-snapshots, and asserts the diff
// against prev equals want; it returns the new snapshot for the next step.
func wantBurst(t *testing.T, a app.App, root string, w *watch.Watcher, prev map[string]app.TaskState, want ...app.Event) map[string]app.TaskState {
	t.Helper()
	waitTick(t, w.C)
	cur := mustSnapshot(t, a, root)
	if got := app.Diff(prev, cur); !slices.Equal(got, want) {
		t.Fatalf("burst events = %+v, want %+v", got, want)
	}
	drainTicks(w.C)
	return cur
}

// streamLines scans r into a channel of lines, closed at EOF.
func streamLines(r io.Reader) <-chan string {
	ch := make(chan string, 64)
	go func() {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			ch <- sc.Text()
		}
		close(ch)
	}()
	return ch
}

// nextWatchLine reads one --agent watch line, checks its RFC3339 timestamp, and
// returns the remaining fields (event, id, field, old, new).
func nextWatchLine(t *testing.T, ch <-chan string) []string {
	t.Helper()
	select {
	case line, ok := <-ch:
		if !ok {
			t.Fatal("watch output closed before an expected line")
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 6 {
			t.Fatalf("watch line %q: %d fields, want 6", line, len(fields))
		}
		if _, err := time.Parse(time.RFC3339, fields[0]); err != nil {
			t.Errorf("watch line %q: bad timestamp: %v", line, err)
		}
		return fields[1:]
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a watch line")
		return nil
	}
}
