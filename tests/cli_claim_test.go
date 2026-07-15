package tests

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/raphaelCamblong/duty/internal/cli"
	"github.com/raphaelCamblong/duty/internal/names"
)

// TestClaimParallel is the concurrency gate: N goroutines each run
// `get next --claim` on a tree of N actionable tasks, and every one must come
// away with a distinct task, the tree left consistent (file and board agree).
// Run under -race, it proves the tree-wide lock serializes the claims.
func TestClaimParallel(t *testing.T) {
	const n = 8
	root := initDuty(t)
	for i := 0; i < n; i++ {
		createTask(t, root, fmt.Sprintf("Task %d", i+1))
	}
	t.Chdir(root)

	codes := make([]int, n)
	outs := make([]string, n)
	errs := make([]string, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			var out, errBuf bytes.Buffer
			codes[i] = cli.Run([]string{"get", "next", "--claim", "--agent"}, strings.NewReader(""), &out, &errBuf, "test")
			outs[i], errs[i] = out.String(), errBuf.String()
		}(i)
	}
	close(start)
	wg.Wait()

	ids := map[string]bool{}
	for i := 0; i < n; i++ {
		if codes[i] != 0 || errs[i] != "" {
			t.Fatalf("claim %d: code=%d stderr=%q", i, codes[i], errs[i])
		}
		id, status := claimID(t, outs[i])
		if status != "in-progress" {
			t.Errorf("claim %d status = %q, want in-progress", i, status)
		}
		if ids[id] {
			t.Errorf("id %s claimed by two goroutines", id)
		}
		ids[id] = true
	}
	if len(ids) != n {
		t.Fatalf("claimed %d distinct ids, want %d", len(ids), n)
	}
	assertAllInProgress(t, root, n)
}

// claimID returns the id and status fields of a `get next --claim --agent`
// record.
func claimID(t *testing.T, out string) (id, status string) {
	t.Helper()
	fields := strings.Split(strings.TrimSuffix(out, "\n"), "\t")
	if len(fields) < 3 {
		t.Fatalf("claim printed %q, want a TSV record", out)
	}
	return fields[0], fields[2]
}

// assertAllInProgress checks every open task reads in-progress in both its file
// and its board row (no drift) — the post-claim consistency invariant.
func assertAllInProgress(t *testing.T, root string, n int) {
	t.Helper()
	code, stdout, stderr := runDuty(t, root, "get", "tasks", "--agent")
	if code != 0 || stderr != "" {
		t.Fatalf("get tasks: code=%d stderr=%q", code, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != n {
		t.Fatalf("get tasks returned %d rows, want %d", len(lines), n)
	}
	for _, line := range lines {
		f := strings.Split(line, "\t")
		if f[2] != "in-progress" {
			t.Errorf("task %s status = %q, want in-progress", f[0], f[2])
		}
		if len(f) >= 5 && f[4] != "" {
			t.Errorf("task %s drift = %q, want file and board in sync", f[0], f[4])
		}
	}
}

func TestClaim(t *testing.T) {
	t.Run("claims the first actionable task and prints it in-progress", func(t *testing.T) {
		root := initDuty(t)
		one := createTask(t, root, "First")
		createTask(t, root, "Second")
		code, stdout, stderr := runDuty(t, root, "get", "next", "--claim")
		if code != 0 || stderr != "" {
			t.Fatalf("claim: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "T-01") || !strings.Contains(stdout, "in-progress") {
			t.Errorf("claim printed %q, want T-01 in-progress", stdout)
		}
		if got := readText(t, filepath.Join(root, one)); !strings.Contains(got, "status: in-progress\n") {
			t.Errorf("claimed file not in-progress: %q", got)
		}
		wantRow := "| [T-01](" + one + ") | First | in-progress |"
		if got := readText(t, filepath.Join(root, "BOARD.md")); !strings.Contains(got, wantRow) {
			t.Errorf("board row not in-progress: %q", got)
		}
	})

	t.Run("a second claim takes the next task", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "First")
		createTask(t, root, "Second")
		mustClaim(t, root, "T-01")
		mustClaim(t, root, "T-02")
	})

	t.Run("nothing actionable: empty output, exit 0, no lock file", func(t *testing.T) {
		root := initDuty(t)
		code, stdout, stderr := runDuty(t, root, "get", "next", "--claim")
		if code != 0 {
			t.Fatalf("claim empty: code=%d stderr=%q", code, stderr)
		}
		if stdout != "" || stderr != "" {
			t.Errorf("claim empty: stdout=%q stderr=%q, want both empty", stdout, stderr)
		}
		if _, err := os.Stat(filepath.Join(root, names.LockFile)); !os.IsNotExist(err) {
			t.Errorf("%s present after an empty claim (err %v), want no lock-file side effect", names.LockFile, err)
		}
	})
}

// mustClaim runs `get next --claim` and asserts it hands back wantID.
func mustClaim(t *testing.T, root, wantID string) {
	t.Helper()
	code, stdout, stderr := runDuty(t, root, "get", "next", "--claim")
	if code != 0 || stderr != "" {
		t.Fatalf("claim: code=%d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, wantID) {
		t.Errorf("claim printed %q, want %s", stdout, wantID)
	}
}

func TestStatusClaimGuard(t *testing.T) {
	t.Run("refuses re-claiming an already in-progress task", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Task")
		mustRun(t, root, "status", "T-01", "in-progress")
		code, stdout, stderr := runDuty(t, root, "status", "T-01", "in-progress")
		if code == 0 {
			t.Fatal("re-claim of an in-progress task succeeded, want refusal")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "already in-progress") || !strings.Contains(stderr, "--force") {
			t.Errorf("stderr = %q, want it to name the claim conflict and --force", stderr)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})

	t.Run("--force takes over an in-progress task", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Task")
		mustRun(t, root, "status", "T-01", "in-progress")
		mustRun(t, root, "status", "T-01", "in-progress", "--force")
		if got := readText(t, filepath.Join(root, name)); !strings.Contains(got, "status: in-progress\n") {
			t.Errorf("file not in-progress after --force take-over: %q", got)
		}
	})

	t.Run("only the in-progress→in-progress transition is guarded", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Task")
		mustRun(t, root, "status", "T-01", "in-progress")
		mustRun(t, root, "status", "T-01", "blocked")
		mustRun(t, root, "status", "T-01", "in-progress")
		mustRun(t, root, "status", "T-01", "done")
		mustRun(t, root, "status", "T-01", "done")
	})
}
