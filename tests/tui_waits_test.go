package tests

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// frameLine returns the first stripped frame line containing s, failing when
// none does.
func frameLine(t *testing.T, frame, s string) string {
	t.Helper()
	for _, l := range strings.Split(ansi.Strip(frame), "\n") {
		if strings.Contains(l, s) {
			return l
		}
	}
	t.Fatalf("frame has no line containing %q:\n%s", s, frame)
	return ""
}

func TestTUIWaitAnnotation(t *testing.T) {
	root := initDuty(t)
	mustDuty(t, root, "create", "task", "Alpha")
	mustDuty(t, root, "create", "task", "Beta")
	mustDuty(t, root, "create", "task", "Waiter",
		"--blocked-by", "T-01", "--blocked-by", "T-02")
	mustDuty(t, root, "status", "T-01", "in-progress") // unmet
	mustDuty(t, root, "status", "T-02", "done")        // met

	t.Run("scan marks only the unmet deps from the snapshot", func(t *testing.T) {
		snap := mustScan(t, root)
		r, ok := scanRow(snap, ".", "T-03")
		if !ok {
			t.Fatal("T-03 not scanned")
		}
		if len(r.Waits) != 1 || r.Waits[0] != "T-01" {
			t.Errorf("T-03 Waits = %v, want [T-01] (T-02 is done)", r.Waits)
		}
		for _, id := range []string{"T-01", "T-02"} {
			if row, _ := scanRow(snap, ".", id); len(row.Waits) != 0 {
				t.Errorf("%s Waits = %v, want none", id, row.Waits)
			}
		}
	})

	t.Run("frame shows the wait annotation on the blocked row", func(t *testing.T) {
		m := newTUIModelSize(t, root, 120, 35)
		line := frameLine(t, m.View().Content, "Waiter")
		if !strings.Contains(line, "waits T-01") {
			t.Errorf("blocked row missing its wait annotation: %q", line)
		}
		if strings.Contains(line, "T-02") {
			t.Errorf("wait annotation names a met dep (T-02): %q", line)
		}
	})
}

func TestTUIArchivedDepCountsMet(t *testing.T) {
	root := initDuty(t)
	mustDuty(t, root, "create", "task", "Done dep")
	mustDuty(t, root, "create", "task", "Waiter", "--blocked-by", "T-01")
	mustDuty(t, root, "status", "T-01", "done")
	mustDuty(t, root, "archive")

	snap := mustScan(t, root)
	if _, ok := scanRow(snap, ".", "T-01"); ok {
		t.Fatal("archived T-01 is still visible in the open snapshot")
	}
	r, ok := scanRow(snap, ".", "T-02")
	if !ok {
		t.Fatal("T-02 not scanned")
	}
	if len(r.Waits) != 0 {
		t.Errorf("T-02 Waits = %v; an archived dep must count as met", r.Waits)
	}
}
