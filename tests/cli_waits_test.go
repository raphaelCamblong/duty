package tests

import (
	"strings"
	"testing"
)

// lineFor returns the get-tasks output line naming task id, "" when none does.
func lineFor(out, id string) string {
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), id+" ") {
			return l
		}
	}
	return ""
}

func TestGetTasksWaits(t *testing.T) {
	root := initDuty(t)
	createTask(t, root, "First dep")  // T-01
	createTask(t, root, "Second dep") // T-02
	mustDuty(t, root, "create", "task", "Waiter",
		"--blocked-by", "T-01", "--blocked-by", "T-02") // T-03

	t.Run("waits names exactly the unmet deps of the blocked task", func(t *testing.T) {
		_, out, _ := runDuty(t, root, "get", "tasks")
		if w := lineFor(out, "T-03"); !strings.Contains(w, "waits T-01,T-02") {
			t.Errorf("T-03 line %q missing waits T-01,T-02", w)
		}
		for _, id := range []string{"T-01", "T-02"} {
			if w := lineFor(out, id); strings.Contains(w, "waits") {
				t.Errorf("%s is actionable but its line shows a wait: %q", id, w)
			}
		}
	})

	t.Run("the annotation shrinks then vanishes as deps finish", func(t *testing.T) {
		mustRun(t, root, "status", "T-01", "done")
		_, out, _ := runDuty(t, root, "get", "tasks")
		if w := lineFor(out, "T-03"); !strings.Contains(w, "waits T-02") || strings.Contains(w, "T-01") {
			t.Errorf("after T-01 done, T-03 line %q should wait on T-02 only", w)
		}

		mustRun(t, root, "status", "T-02", "done")
		_, out, _ = runDuty(t, root, "get", "tasks")
		if w := lineFor(out, "T-03"); strings.Contains(w, "waits") {
			t.Errorf("with every dep done, T-03 must be actionable, got %q", w)
		}
	})
}

func TestGetTaskBlockedByStatuses(t *testing.T) {
	root := initDuty(t)
	createTask(t, root, "One") // T-01
	createTask(t, root, "Two") // T-02
	mustDuty(t, root, "create", "task", "Main",
		"--blocked-by", "T-01", "--blocked-by", "T-02") // T-03
	mustRun(t, root, "status", "T-01", "in-progress")
	mustRun(t, root, "status", "T-02", "done")

	t.Run("human annotates each blocked-by id with its status", func(t *testing.T) {
		_, out, _ := runDuty(t, root, "get", "task", "T-03")
		if !strings.Contains(out, "blocked-by: T-01 (in-progress), T-02 (done)") {
			t.Errorf("get task T-03 missing per-dep statuses:\n%s", out)
		}
	})

	t.Run("no deps still reads none", func(t *testing.T) {
		_, out, _ := runDuty(t, root, "get", "task", "T-01")
		if !strings.Contains(out, "blocked-by: none") {
			t.Errorf("get task T-01 blocked-by should read none:\n%s", out)
		}
	})

	t.Run("--agent TSV keeps the plain comma-joined blocked-by field", func(t *testing.T) {
		_, out, _ := runDuty(t, root, "get", "task", "T-03", "--agent")
		fields := strings.Split(strings.TrimRight(out, "\n"), "\t")
		if len(fields) != 10 {
			t.Fatalf("record %q: got %d fields, want 10", out, len(fields))
		}
		if fields[6] != "T-01,T-02" {
			t.Errorf("blocked-by field = %q, want unannotated T-01,T-02", fields[6])
		}
	})
}
