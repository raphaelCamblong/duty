package tests

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/raphaelCamblong/duty/internal/cli"
)

// claimedBy reports whether the task file at path records name as its claimer.
func claimedBy(t *testing.T, path, name string) bool {
	t.Helper()
	return strings.Contains(readText(t, path), "claimed-by: "+name+"\n")
}

func TestClaimIdentity(t *testing.T) {
	t.Run("--as records the claimer, surfaces it, and clears it on transition", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Hold me")
		path := filepath.Join(root, name)

		mustRun(t, root, "status", "T-01", "in-progress", "--as", "sonnet-2")
		if !claimedBy(t, path, "sonnet-2") {
			t.Errorf("file missing claimed-by after --as claim:\n%s", readText(t, path))
		}

		_, human, _ := runDuty(t, root, "get", "task", "T-01")
		if !strings.Contains(human, "claimed-by: sonnet-2") {
			t.Errorf("get task human missing the claimer line:\n%s", human)
		}

		mustRun(t, root, "status", "T-01", "done")
		if strings.Contains(readText(t, path), "claimed-by:") {
			t.Errorf("done did not clear the claim:\n%s", readText(t, path))
		}
	})

	t.Run("blocked also clears the claim (only means currently holds it)", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Park me")
		path := filepath.Join(root, name)
		mustRun(t, root, "status", "T-01", "in-progress", "--as", "haiku")
		mustRun(t, root, "status", "T-01", "blocked")
		if strings.Contains(readText(t, path), "claimed-by:") {
			t.Errorf("blocked did not clear the claim:\n%s", readText(t, path))
		}
	})

	t.Run("DUTY_AGENT is the fallback and --as overrides it", func(t *testing.T) {
		root := initDuty(t)
		nameA := createTask(t, root, "Env one")
		nameB := createTask(t, root, "Flag two")
		t.Setenv("DUTY_AGENT", "env-agent")

		mustRun(t, root, "status", "T-01", "in-progress")
		if !claimedBy(t, filepath.Join(root, nameA), "env-agent") {
			t.Errorf("env fallback not recorded:\n%s", readText(t, filepath.Join(root, nameA)))
		}

		mustRun(t, root, "status", "T-02", "in-progress", "--as", "flag-agent")
		if !claimedBy(t, filepath.Join(root, nameB), "flag-agent") {
			t.Errorf("--as did not override DUTY_AGENT:\n%s", readText(t, filepath.Join(root, nameB)))
		}
	})

	t.Run("an unnamed claim writes no claimed-by line (no regression)", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "No name")
		mustRun(t, root, "status", "T-01", "in-progress")
		if strings.Contains(readText(t, filepath.Join(root, name)), "claimed-by:") {
			t.Errorf("unnamed claim wrote a claimed-by line:\n%s", readText(t, filepath.Join(root, name)))
		}
	})

	t.Run("claim then unclaim leaves the tree byte-identical", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Round trip")
		before := hashTree(t, root)

		mustRun(t, root, "status", "T-01", "in-progress", "--as", "sonnet-2")
		mustRun(t, root, "status", "T-01", "todo")
		if got := hashTree(t, root); got != before {
			t.Errorf("tree hash after claim+unclaim = %s, want %s", got, before)
		}
	})

	t.Run("the --force refusal names the current holder", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Contested")
		path := filepath.Join(root, name)
		mustRun(t, root, "status", "T-01", "in-progress", "--as", "sonnet-2")

		code, stdout, stderr := runDuty(t, root, "status", "T-01", "in-progress", "--as", "other")
		if code == 0 {
			t.Fatal("re-claim without --force succeeded, want refusal")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "claimed by sonnet-2") || !strings.Contains(stderr, "--force") {
			t.Errorf("stderr = %q, want it to name the holder and --force", stderr)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if !claimedBy(t, path, "sonnet-2") {
			t.Errorf("refused re-claim changed the holder:\n%s", readText(t, path))
		}

		mustRun(t, root, "status", "T-01", "in-progress", "--as", "other", "--force")
		if !claimedBy(t, path, "other") {
			t.Errorf("--force take-over did not update the holder:\n%s", readText(t, path))
		}
	})

	t.Run("get task --agent trails the claimer as a new field", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Trailing")
		mustRun(t, root, "status", "T-01", "in-progress", "--as", "sonnet-2")

		_, agent, _ := runDuty(t, root, "get", "task", "T-01", "--agent")
		fields := strings.Split(strings.TrimRight(agent, "\n"), "\t")
		if len(fields) != 10 {
			t.Fatalf("record %q: got %d fields, want 10", agent, len(fields))
		}
		if fields[9] != "sonnet-2" {
			t.Errorf("trailing claimed-by field = %q, want sonnet-2", fields[9])
		}
	})

	t.Run("get tasks human shows in-progress · claimer", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Listed")
		mustRun(t, root, "status", "T-01", "in-progress", "--as", "sonnet-2")

		_, human, _ := runDuty(t, root, "get", "tasks")
		if !strings.Contains(human, "in-progress · sonnet-2") {
			t.Errorf("get tasks human missing the claimer:\n%s", human)
		}
	})
}

// TestClaimIdentityParallel is the parallel-named-claims gate: N goroutines each
// claim as a distinct agent; every one comes away with a distinct task whose
// file records that goroutine's name.
func TestClaimIdentityParallel(t *testing.T) {
	const n = 6
	root := initDuty(t)
	for i := 0; i < n; i++ {
		createTask(t, root, fmt.Sprintf("Task %d", i+1))
	}
	t.Chdir(root)

	type claim struct{ id, claimer, path string }
	claims := make([]claim, n)
	codes := make([]int, n)
	errs := make([]string, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			var out, errBuf bytes.Buffer
			name := fmt.Sprintf("agent-%d", i)
			codes[i] = cli.Run([]string{"get", "next", "--claim", "--as", name, "--agent"},
				strings.NewReader(""), &out, &errBuf, "test")
			errs[i] = errBuf.String()
			if f := strings.Split(strings.TrimRight(out.String(), "\n"), "\t"); len(f) == 10 {
				claims[i] = claim{id: f[0], claimer: f[9], path: f[7]}
			}
		}(i)
	}
	close(start)
	wg.Wait()

	ids := map[string]bool{}
	for i := 0; i < n; i++ {
		if codes[i] != 0 || errs[i] != "" {
			t.Fatalf("claim %d: code=%d stderr=%q", i, codes[i], errs[i])
		}
		c := claims[i]
		want := fmt.Sprintf("agent-%d", i)
		if c.claimer != want {
			t.Errorf("goroutine %d claimed as %q, want %q", i, c.claimer, want)
		}
		if ids[c.id] {
			t.Errorf("id %s claimed by two goroutines", c.id)
		}
		ids[c.id] = true
		if !claimedBy(t, c.path, want) {
			t.Errorf("file for %s missing claimed-by %q:\n%s", c.id, want, readText(t, c.path))
		}
	}
	if len(ids) != n {
		t.Fatalf("claimed %d distinct ids, want %d", len(ids), n)
	}
}

func TestClaimShownInTUI(t *testing.T) {
	root := initDuty(t)
	mustDuty(t, root, "create", "task", "Alpha task")
	mustDuty(t, root, "status", "T-01", "in-progress", "--as", "sonnet-2")

	r, ok := scanRow(mustScan(t, root), ".", "T-01")
	if !ok {
		t.Fatal("T-01 not in snapshot")
	}
	if r.ClaimedBy != "sonnet-2" {
		t.Errorf("scanned ClaimedBy = %q, want sonnet-2", r.ClaimedBy)
	}

	m := newTUIModelSize(t, root, 120, 35)
	browse := m.View().Content
	if !strings.Contains(browse, "sonnet-2") {
		t.Errorf("board frame missing the holder next to the row:\n%s", browse)
	}
	t.Logf("board 120x35 (row shows the holder):\n%s", browse)

	m, _ = press(t, m, "enter") // open T-01's preview
	if m.DetailID() != "T-01" {
		t.Fatalf("enter did not open T-01: detail=%q", m.DetailID())
	}
	open := m.View().Content
	if !strings.Contains(open, "sonnet-2") {
		t.Errorf("preview header missing the holder name:\n%s", open)
	}
	t.Logf("preview 120x35 (header shows the holder):\n%s", open)
}
