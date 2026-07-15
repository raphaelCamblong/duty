package tests

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/raphaelCamblong/duty/internal/cli"
)

// mustRunStdin runs a duty command fed input on stdin that has to succeed
// quietly.
func mustRunStdin(t *testing.T, dir, input string, args ...string) {
	t.Helper()
	code, stdout, stderr := runDutyStdin(t, dir, input, args...)
	if code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("duty %v: code=%d stdout=%q stderr=%q", args, code, stdout, stderr)
	}
}

func TestGetTaskSection(t *testing.T) {
	t.Run("prints a section body, trimmed of its framing blank lines", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Section me")
		mustRunStdin(t, root, "The stated outcome.\n", "set", "T-01", "goal")
		code, stdout, stderr := runDuty(t, root, "get", "task", "T-01", "--section", "goal")
		if code != 0 || stderr != "" {
			t.Fatalf("get task --section: code=%d stderr=%q", code, stderr)
		}
		if stdout != "The stated outcome.\n" {
			t.Errorf("stdout = %q, want the goal body", stdout)
		}
	})

	t.Run("matches the heading case-insensitively", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Section me")
		mustRunStdin(t, root, "Body.\n", "set", "T-01", "goal")
		code, stdout, stderr := runDuty(t, root, "get", "task", "T-01", "--section", "GOAL")
		if code != 0 || stderr != "" || stdout != "Body.\n" {
			t.Fatalf("get task --section GOAL: code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})

	t.Run("unknown section exits 1 naming the id", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Section me")
		code, stdout, stderr := runDuty(t, root, "get", "task", "T-01", "--section", "design")
		if code != 1 {
			t.Fatalf("code = %d, want 1", code)
		}
		oneLine(t, "stderr", stderr)
		if stderr != "no section \"design\" in T-01\n" {
			t.Errorf("stderr = %q, want the missing-section message", stderr)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})

	t.Run("rejects archived ids", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old-work.md")
		code, _, stderr := runDuty(t, root, "get", "task", "T-90", "--section", "goal")
		if code == 0 {
			t.Fatal("get task --section on an archived id succeeded")
		}
		if !strings.Contains(stderr, "archived") {
			t.Errorf("stderr = %q, want it to say archived", stderr)
		}
	})
}

func TestSetSection(t *testing.T) {
	t.Run("replaces a section body from stdin, touching nothing else", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Set me")
		before := readText(t, filepath.Join(root, name))
		mustRunStdin(t, root, "The stated outcome.\n", "set", "T-01", "goal")
		want := replaceOnce(t, before, "## Goal\n\n", "## Goal\nThe stated outcome.\n\n")
		if got := readText(t, filepath.Join(root, name)); got != want {
			t.Errorf("set goal =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("replacing twice overwrites, never accumulates", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Set me")
		mustRunStdin(t, root, "First.\n", "set", "T-01", "goal")
		mustRunStdin(t, root, "Second.\n", "set", "T-01", "goal")
		got := readText(t, filepath.Join(root, name))
		if strings.Contains(got, "First.") {
			t.Errorf("old goal survived a second set: %q", got)
		}
		if strings.Count(got, "Second.") != 1 {
			t.Errorf("want one goal body, got %q", got)
		}
	})

	t.Run("creates a missing section before Report", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Set me")
		mustRunStdin(t, root, "Ports and adapters.\n", "set", "T-01", "Design")
		got := readText(t, filepath.Join(root, name))
		designAt := strings.Index(got, "## Design")
		reportAt := strings.Index(got, "## Report")
		if designAt < 0 || reportAt < designAt {
			t.Errorf("want ## Design created above ## Report, got %q", got)
		}
	})

	t.Run("refuses empty stdin, changing nothing", func(t *testing.T) {
		for _, input := range []string{"", "\n \t\n"} {
			root := initDuty(t)
			name := createTask(t, root, "Set me")
			before := readText(t, filepath.Join(root, name))
			code, stdout, stderr := runDutyStdin(t, root, input, "set", "T-01", "goal")
			if code == 0 {
				t.Fatalf("set with stdin %q succeeded, want refusal", input)
			}
			oneLine(t, "stderr", stderr)
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if readText(t, filepath.Join(root, name)) != before {
				t.Error("task file changed by refused set")
			}
		}
	})

	t.Run("rejects archived ids", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old-work.md")
		code, _, stderr := runDutyStdin(t, root, "Late edit.\n", "set", "T-90", "goal")
		if code == 0 {
			t.Fatal("set on an archived id succeeded")
		}
		if !strings.Contains(stderr, "archived") {
			t.Errorf("stderr = %q, want it to say archived", stderr)
		}
	})

	t.Run("argument validation", func(t *testing.T) {
		for _, args := range [][]string{{"set"}, {"set", "T-01"}, {"set", "T-01", "goal", "extra"}} {
			root := initDuty(t)
			createTask(t, root, "Set me")
			code, _, stderr := runDutyStdin(t, root, "text\n", args...)
			if code == 0 {
				t.Errorf("duty %v succeeded, want usage error", args)
			}
			oneLine(t, "stderr", stderr)
		}
	})
}

func TestGatesCLI(t *testing.T) {
	t.Run("lists gates 1-based, bare and with the list word", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Gate me")
		mustRun(t, root, "gates", "add", "T-01", "build passes")
		mustRun(t, root, "gates", "add", "T-01", "tests green")
		mustRun(t, root, "gates", "check", "T-01", "1")
		code, stdout, stderr := runDuty(t, root, "gates", "T-01")
		if code != 0 || stderr != "" {
			t.Fatalf("gates: code=%d stderr=%q", code, stderr)
		}
		if stdout != "1 [x] build passes\n2 [ ] tests green\n" {
			t.Errorf("gates =\n%q", stdout)
		}
		_, listOut, _ := runDuty(t, root, "gates", "T-01", "list")
		if listOut != stdout {
			t.Errorf("gates list = %q, want same as bare gates %q", listOut, stdout)
		}
	})

	t.Run("--agent emits index/done/text TSV", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Gate me")
		mustRun(t, root, "gates", "add", "T-01", "build passes")
		mustRun(t, root, "gates", "add", "T-01", "tests green")
		mustRun(t, root, "gates", "check", "T-01", "2")
		code, stdout, stderr := runDuty(t, root, "gates", "T-01", "--agent")
		if code != 0 || stderr != "" {
			t.Fatalf("gates --agent: code=%d stderr=%q", code, stderr)
		}
		if stdout != "1\tfalse\tbuild passes\n2\ttrue\ttests green\n" {
			t.Errorf("gates --agent =\n%q", stdout)
		}
	})

	t.Run("check and uncheck flip exactly one checkbox line", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Gate me")
		mustRun(t, root, "gates", "add", "T-01", "alpha")
		mustRun(t, root, "gates", "add", "T-01", "beta")
		before := readText(t, filepath.Join(root, name))
		mustRun(t, root, "gates", "check", "T-01", "2")
		want := replaceOnce(t, before, "- [ ] beta", "- [x] beta")
		if got := readText(t, filepath.Join(root, name)); got != want {
			t.Errorf("gates check =\n%q\nwant:\n%q", got, want)
		}
		mustRun(t, root, "gates", "uncheck", "T-01", "2")
		if got := readText(t, filepath.Join(root, name)); got != before {
			t.Errorf("uncheck did not restore the line:\n got %q\nwant %q", got, before)
		}
	})

	t.Run("check past the last gate exits 1, changing nothing", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Gate me")
		mustRun(t, root, "gates", "add", "T-01", "only")
		before := readText(t, filepath.Join(root, name))
		code, stdout, stderr := runDuty(t, root, "gates", "check", "T-01", "9")
		if code != 1 {
			t.Fatalf("code = %d, want 1", code)
		}
		oneLine(t, "stderr", stderr)
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if readText(t, filepath.Join(root, name)) != before {
			t.Error("task file changed by out-of-range check")
		}
	})

	t.Run("add creates the Gates section on a fresh task", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Gate me")
		mustRun(t, root, "gates", "add", "T-01", "first gate")
		code, stdout, stderr := runDuty(t, root, "gates", "T-01")
		if code != 0 || stderr != "" {
			t.Fatalf("gates: code=%d stderr=%q", code, stderr)
		}
		if stdout != "1 [ ] first gate\n" {
			t.Errorf("gates =\n%q", stdout)
		}
	})

	t.Run("argument validation", func(t *testing.T) {
		for _, args := range [][]string{
			{"gates"},
			{"gates", "add", "T-01"},
			{"gates", "check", "T-01"},
			{"gates", "check", "T-01", "zero"},
			{"gates", "check", "T-01", "0"},
			{"gates", "uncheck", "T-01", "-1"},
		} {
			root := initDuty(t)
			createTask(t, root, "Gate me")
			code, _, stderr := runDuty(t, root, args...)
			if code == 0 {
				t.Errorf("duty %v succeeded, want usage error", args)
			}
			oneLine(t, "stderr", stderr)
		}
	})
}

// TestAuthoringFlowNoEditor is the task's headline gate: a task authored end to
// end with only the CLI — create, set goal, set scope, add two gates, check
// one, then read a section and the gate list back — no editor ever involved.
func TestAuthoringFlowNoEditor(t *testing.T) {
	root := initDuty(t)
	createTask(t, root, "Editor-free task")
	mustRunStdin(t, root, "Ship the flow.\n", "set", "T-01", "goal")
	mustRunStdin(t, root, "Domain, app, cli.\n", "set", "T-01", "scope")
	mustRun(t, root, "gates", "add", "T-01", "build passes")
	mustRun(t, root, "gates", "add", "T-01", "tests green")
	mustRun(t, root, "gates", "check", "T-01", "1")

	code, goal, stderr := runDuty(t, root, "get", "task", "T-01", "--section", "goal")
	if code != 0 || stderr != "" || goal != "Ship the flow.\n" {
		t.Fatalf("get section goal: code=%d stdout=%q stderr=%q", code, goal, stderr)
	}
	if _, scope, _ := runDuty(t, root, "get", "task", "T-01", "--section", "scope"); scope != "Domain, app, cli.\n" {
		t.Errorf("scope = %q, want the set body", scope)
	}
	code, gates, stderr := runDuty(t, root, "gates", "T-01")
	if code != 0 || stderr != "" {
		t.Fatalf("gates: code=%d stderr=%q", code, stderr)
	}
	if gates != "1 [x] build passes\n2 [ ] tests green\n" {
		t.Errorf("gates =\n%q", gates)
	}
}

// TestGatesCheckParallel is the concurrency gate: N goroutines each check a
// distinct gate on one task at once. The tree-wide lock serializes the
// read-modify-write, so under -race every check must land — all N gates ticked.
func TestGatesCheckParallel(t *testing.T) {
	const n = 8
	root := initDuty(t)
	createTask(t, root, "Many gates")
	for i := 0; i < n; i++ {
		mustRun(t, root, "gates", "add", "T-01", fmt.Sprintf("gate %d", i+1))
	}
	t.Chdir(root)

	var wg sync.WaitGroup
	codes := make([]int, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			var out, errBuf bytes.Buffer
			codes[i] = cli.Run([]string{"gates", "check", "T-01", strconv.Itoa(i + 1)}, strings.NewReader(""), &out, &errBuf, "test")
		}(i)
	}
	close(start)
	wg.Wait()

	for i, code := range codes {
		if code != 0 {
			t.Errorf("check %d: code = %d, want 0", i+1, code)
		}
	}
	code, stdout, stderr := runDuty(t, root, "gates", "T-01", "--agent")
	if code != 0 || stderr != "" {
		t.Fatalf("gates --agent: code=%d stderr=%q", code, stderr)
	}
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if f := strings.Split(line, "\t"); f[1] != "true" {
			t.Errorf("gate %q not ticked after parallel checks — a write was lost", line)
		}
	}
}
