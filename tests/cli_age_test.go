package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mtimeRFC3339 returns path's modification time formatted as RFC3339, matching
// the trailing --agent field.
func mtimeRFC3339(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.ModTime().Format(time.RFC3339)
}

// backdate sets path's mtime to d before now (whole seconds so the bucketed age
// is deterministic).
func backdate(t *testing.T, path string, d time.Duration) {
	t.Helper()
	when := time.Now().Add(-d).Truncate(time.Second)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}

func TestTaskAgeReads(t *testing.T) {
	t.Run("get task shows an updated line and appends the RFC3339 mtime", func(t *testing.T) {
		root := initDuty(t)
		path := filepath.Join(root, createTask(t, root, "Aged task"))
		backdate(t, path, 2*time.Hour)

		_, human, _ := runDuty(t, root, "get", "task", "T-01")
		if !strings.Contains(human, "updated:") || !strings.Contains(human, "2h ago") {
			t.Errorf("human get task missing the updated age:\n%s", human)
		}

		_, agent, _ := runDuty(t, root, "get", "task", "T-01", "--agent")
		fields := strings.Split(strings.TrimRight(agent, "\n"), "\t")
		if len(fields) != 10 {
			t.Fatalf("record %q: got %d fields, want 10", agent, len(fields))
		}
		if got, want := fields[8], mtimeRFC3339(t, path); got != want {
			t.Errorf("mtime field = %q, want the RFC3339 mtime %q", got, want)
		}
		if fields[9] != "" {
			t.Errorf("claimed-by field = %q, want empty (unclaimed)", fields[9])
		}
	})

	t.Run("get tasks gains a trailing age column and the RFC3339 mtime", func(t *testing.T) {
		root := initDuty(t)
		path := filepath.Join(root, createTask(t, root, "Aged task"))
		backdate(t, path, 3*24*time.Hour)

		_, human, _ := runDuty(t, root, "get", "tasks")
		if !strings.Contains(human, "3d ago") {
			t.Errorf("human get tasks missing the age column:\n%s", human)
		}

		_, agent, _ := runDuty(t, root, "get", "tasks", "--agent")
		fields := strings.Split(strings.TrimRight(agent, "\n"), "\t")
		if len(fields) != 6 {
			t.Fatalf("record %q: got %d fields, want 6", agent, len(fields))
		}
		if got, want := fields[5], mtimeRFC3339(t, path); got != want {
			t.Errorf("trailing field = %q, want the RFC3339 mtime %q", got, want)
		}
	})

	t.Run("get next appends the RFC3339 mtime as its trailing field", func(t *testing.T) {
		root := initDuty(t)
		path := filepath.Join(root, createTask(t, root, "Aged task"))
		backdate(t, path, 90*time.Second)

		_, human, _ := runDuty(t, root, "get", "next")
		if !strings.Contains(human, "1m ago") {
			t.Errorf("human get next missing the updated age:\n%s", human)
		}

		_, agent, _ := runDuty(t, root, "get", "next", "--agent")
		fields := strings.Split(strings.TrimRight(agent, "\n"), "\t")
		if len(fields) != 10 {
			t.Fatalf("record %q: got %d fields, want 10", agent, len(fields))
		}
		if got, want := fields[8], mtimeRFC3339(t, path); got != want {
			t.Errorf("mtime field = %q, want the RFC3339 mtime %q", got, want)
		}
		if fields[9] != "" {
			t.Errorf("claimed-by field = %q, want empty (unclaimed)", fields[9])
		}
	})
}
