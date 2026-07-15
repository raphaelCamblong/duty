package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

// bodyAfterFrontmatter returns everything below a task file's frontmatter
// block — the bytes get task --body must print verbatim.
func bodyAfterFrontmatter(t *testing.T, content string) string {
	t.Helper()
	parts := strings.SplitN(content, "---\n", 3)
	if len(parts) != 3 {
		t.Fatalf("no frontmatter block in %q", content)
	}
	return parts[2]
}

func TestReportStatus(t *testing.T) {
	t.Run("appends the report and flips status (file + board) in one call, byte-identical to report then status", func(t *testing.T) {
		oneShot := initDuty(t)
		nameA := createTask(t, oneShot, "End me")
		mustRunStdin(t, oneShot, "All gates green.\n", "report", "T-01", "--status", "done")

		twoCall := initDuty(t)
		nameB := createTask(t, twoCall, "End me")
		mustRunStdin(t, twoCall, "All gates green.\n", "report", "T-01")
		mustRun(t, twoCall, "status", "T-01", "done")

		if a, b := readText(t, filepath.Join(oneShot, nameA)), readText(t, filepath.Join(twoCall, nameB)); a != b {
			t.Errorf("one-shot task file =\n%q\nwant (report then status):\n%q", a, b)
		}
		if a, b := readText(t, filepath.Join(oneShot, "BOARD.md")), readText(t, filepath.Join(twoCall, "BOARD.md")); a != b {
			t.Errorf("one-shot board =\n%q\nwant (report then status):\n%q", a, b)
		}
	})

	t.Run("an unknown status lands neither the report nor the status", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Untouched")
		taskBefore := readText(t, filepath.Join(root, name))
		boardBefore := readText(t, filepath.Join(root, "BOARD.md"))

		code, stdout, stderr := runDutyStdin(t, root, "Some words.\n", "report", "T-01", "--status", "cancelled")
		if code == 0 {
			t.Fatal("report --status cancelled succeeded, want refusal")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "cancelled") {
			t.Errorf("stderr = %q, want it to name the bad status", stderr)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if readText(t, filepath.Join(root, name)) != taskBefore {
			t.Error("task file changed by refused report --status: the report leaked")
		}
		if readText(t, filepath.Join(root, "BOARD.md")) != boardBefore {
			t.Error("board changed by refused report --status")
		}
	})

	t.Run("empty stdin refuses and flips no status", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Untouched")
		taskBefore := readText(t, filepath.Join(root, name))
		boardBefore := readText(t, filepath.Join(root, "BOARD.md"))

		code, _, stderr := runDutyStdin(t, root, "\n \t\n", "report", "T-01", "--status", "done")
		if code == 0 {
			t.Fatal("report --status done with blank stdin succeeded, want refusal")
		}
		oneLine(t, "stderr", stderr)
		if readText(t, filepath.Join(root, name)) != taskBefore {
			t.Error("task file changed: status flipped despite the blank-report refusal")
		}
		if readText(t, filepath.Join(root, "BOARD.md")) != boardBefore {
			t.Error("board changed by refused report --status")
		}
	})

	t.Run("the claim guard blocks --status in-progress, landing neither, until --force", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Claimed")
		mustRun(t, root, "status", "T-01", "in-progress")
		taskBefore := readText(t, filepath.Join(root, name))
		boardBefore := readText(t, filepath.Join(root, "BOARD.md"))

		code, _, stderr := runDutyStdin(t, root, "Taking over.\n", "report", "T-01", "--status", "in-progress")
		if code == 0 {
			t.Fatal("report --status in-progress on a live claim succeeded without --force")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "already in-progress") {
			t.Errorf("stderr = %q, want the claim-guard message", stderr)
		}
		if readText(t, filepath.Join(root, name)) != taskBefore {
			t.Error("task file changed by the refused claim: the report leaked")
		}
		if readText(t, filepath.Join(root, "BOARD.md")) != boardBefore {
			t.Error("board changed by the refused claim")
		}

		mustRunStdin(t, root, "Taking over.\n", "report", "T-01", "--status", "in-progress", "--force")
		if got := readText(t, filepath.Join(root, name)); !strings.HasSuffix(got, "Taking over.\n") {
			t.Errorf("forced report did not land: %q", got)
		}
	})

	t.Run("rejects archived ids", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old-work.md")
		code, _, stderr := runDutyStdin(t, root, "Late words.\n", "report", "T-90", "--status", "done")
		if code == 0 {
			t.Fatal("report --status on an archived id succeeded")
		}
		oneLine(t, "stderr", stderr)
	})
}

func TestGatesCheckAll(t *testing.T) {
	t.Run("check --all ticks every gate in one write, surgically", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Many gates")
		mustRun(t, root, "gates", "add", "T-01", "alpha")
		mustRun(t, root, "gates", "add", "T-01", "beta")
		mustRun(t, root, "gates", "add", "T-01", "gamma")
		mustRun(t, root, "gates", "check", "T-01", "2")
		before := readText(t, filepath.Join(root, name))

		mustRun(t, root, "gates", "check", "T-01", "--all")
		want := strings.ReplaceAll(before, "- [ ] ", "- [x] ")
		if got := readText(t, filepath.Join(root, name)); got != want {
			t.Errorf("check --all =\n%q\nwant every box ticked and nothing else touched:\n%q", got, want)
		}
	})

	t.Run("uncheck --all unticks every gate", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Many gates")
		mustRun(t, root, "gates", "add", "T-01", "alpha")
		mustRun(t, root, "gates", "add", "T-01", "beta")
		mustRun(t, root, "gates", "check", "T-01", "--all")
		mustRun(t, root, "gates", "uncheck", "T-01", "--all")
		code, stdout, stderr := runDuty(t, root, "gates", "T-01", "--agent")
		if code != 0 || stderr != "" {
			t.Fatalf("gates --agent: code=%d stderr=%q", code, stderr)
		}
		if stdout != "1\tfalse\talpha\n2\tfalse\tbeta\n" {
			t.Errorf("gates --agent =\n%q, want both unticked", stdout)
		}
	})

	t.Run("--all on a gateless task is a clean no-op", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "No gates")
		before := readText(t, filepath.Join(root, name))
		mustRun(t, root, "gates", "check", "T-01", "--all")
		if got := readText(t, filepath.Join(root, name)); got != before {
			t.Errorf("check --all changed a gateless task:\n got %q\nwant %q", got, before)
		}
	})

	t.Run("argument validation", func(t *testing.T) {
		for _, args := range [][]string{
			{"gates", "check", "--all"},
			{"gates", "check", "T-01", "--all", "2"},
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

func TestGatesAddVariadic(t *testing.T) {
	t.Run("appends several gates in order in one call", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Gate me")
		mustRun(t, root, "gates", "add", "T-01", "alpha", "beta", "gamma")
		code, stdout, stderr := runDuty(t, root, "gates", "T-01")
		if code != 0 || stderr != "" {
			t.Fatalf("gates: code=%d stderr=%q", code, stderr)
		}
		if stdout != "1 [ ] alpha\n2 [ ] beta\n3 [ ] gamma\n" {
			t.Errorf("gates =\n%q, want the three in order", stdout)
		}
	})

	t.Run("a later variadic add appends after the existing gates", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Gate me")
		mustRun(t, root, "gates", "add", "T-01", "one", "two")
		mustRun(t, root, "gates", "add", "T-01", "three", "four")
		_, stdout, _ := runDuty(t, root, "gates", "T-01")
		if stdout != "1 [ ] one\n2 [ ] two\n3 [ ] three\n4 [ ] four\n" {
			t.Errorf("gates =\n%q, want the four in add order", stdout)
		}
	})

	t.Run("a single text still works", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Gate me")
		mustRun(t, root, "gates", "add", "T-01", "only")
		_, stdout, _ := runDuty(t, root, "gates", "T-01")
		if stdout != "1 [ ] only\n" {
			t.Errorf("gates =\n%q", stdout)
		}
	})

	t.Run("argument validation", func(t *testing.T) {
		for _, args := range [][]string{
			{"gates", "add", "T-01"},
			{"gates", "add", "T-01", ""},
			{"gates", "add", "T-01", "ok", ""},
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

func TestGetTaskBody(t *testing.T) {
	t.Run("prints the whole body below the frontmatter, byte-identical to the file", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Read me whole")
		mustRunStdin(t, root, "The outcome.\n", "set", "T-01", "goal")
		mustRun(t, root, "gates", "add", "T-01", "build passes")

		want := bodyAfterFrontmatter(t, readText(t, filepath.Join(root, name)))
		code, stdout, stderr := runDuty(t, root, "get", "task", "T-01", "--body")
		if code != 0 || stderr != "" {
			t.Fatalf("get task --body: code=%d stderr=%q", code, stderr)
		}
		if stdout != want {
			t.Errorf("get task --body =\n%q\nwant the file body:\n%q", stdout, want)
		}
	})

	t.Run("--body, --section and --agent are mutually exclusive", func(t *testing.T) {
		for _, args := range [][]string{
			{"get", "task", "T-01", "--body", "--section", "goal"},
			{"get", "task", "T-01", "--body", "--agent"},
		} {
			root := initDuty(t)
			createTask(t, root, "Read me")
			code, stdout, stderr := runDuty(t, root, args...)
			if code != 1 {
				t.Fatalf("duty %v: code = %d, want 1", args, code)
			}
			oneLine(t, "stderr", stderr)
			if stdout != "" {
				t.Errorf("duty %v: stdout = %q, want empty", args, stdout)
			}
		}
	})

	t.Run("rejects archived ids", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old-work.md")
		code, _, stderr := runDuty(t, root, "get", "task", "T-90", "--body")
		if code == 0 {
			t.Fatal("get task --body on an archived id succeeded")
		}
		if !strings.Contains(stderr, "archived") {
			t.Errorf("stderr = %q, want it to say archived", stderr)
		}
	})
}

// TestAgentLifecycleFourCalls is the task's headline gate: an agent takes a
// task from authored to done in exactly four CLI calls — create task --body,
// get next --claim, gates check --all, report --status done — and the tree
// reflects every step.
func TestAgentLifecycleFourCalls(t *testing.T) {
	root := initDuty(t)

	body := "## Goal\nShip it.\n\n## Gates\n- [ ] build passes\n- [ ] tests green\n\n## Report\n"
	code, createOut, stderr := runDutyStdin(t, root, body, "create", "task", "The whole loop", "--body")
	if code != 0 || stderr != "" {
		t.Fatalf("1/4 create --body: code=%d stderr=%q", code, stderr)
	}
	_, createPath := splitCreateOutput(t, createOut)
	name := filepath.Base(createPath)

	code, nextOut, stderr := runDuty(t, root, "get", "next", "--claim", "--agent")
	if code != 0 || stderr != "" {
		t.Fatalf("2/4 get next --claim: code=%d stderr=%q", code, stderr)
	}
	if f := strings.Split(strings.TrimRight(nextOut, "\n"), "\t"); f[0] != "T-01" || f[2] != "in-progress" {
		t.Fatalf("get next --claim = %q, want T-01 claimed in-progress", nextOut)
	}

	if code, _, stderr := runDuty(t, root, "gates", "check", "T-01", "--all"); code != 0 || stderr != "" {
		t.Fatalf("3/4 gates check --all: code=%d stderr=%q", code, stderr)
	}

	if code, _, stderr := runDutyStdin(t, root, "Shipped: build and tests green.\n", "report", "T-01", "--status", "done"); code != 0 || stderr != "" {
		t.Fatalf("4/4 report --status done: code=%d stderr=%q", code, stderr)
	}

	file := readText(t, filepath.Join(root, name))
	if !strings.Contains(file, "\nstatus: done\n") {
		t.Errorf("task not done in the file: %q", file)
	}
	if strings.Contains(file, "- [ ]") {
		t.Errorf("a gate stayed unticked after check --all: %q", file)
	}
	if !strings.HasSuffix(file, "Shipped: build and tests green.\n") {
		t.Errorf("report not recorded: %q", file)
	}
	if got := readText(t, filepath.Join(root, "BOARD.md")); !strings.Contains(got, "| done |") {
		t.Errorf("board row not done: %q", got)
	}
}
