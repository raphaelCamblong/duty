package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

// bodyBelowH1 returns everything from the first "## " section heading onward —
// the task body, id- and H1-independent, so two tasks with different ids are
// comparable below their headings.
func bodyBelowH1(t *testing.T, content string) string {
	t.Helper()
	i := strings.Index(content, "\n## ")
	if i < 0 {
		t.Fatalf("no section heading in %q", content)
	}
	return content[i:]
}

func TestCreateTaskBody(t *testing.T) {
	t.Run("one-shot --body is byte-identical below the H1 to N set calls", func(t *testing.T) {
		root := initDuty(t)
		nameB := createTask(t, root, "Section by section")
		mustRunStdin(t, root, "The outcome.\n", "set", "T-01", "goal")
		mustRunStdin(t, root, "Read the spec.\n", "set", "T-01", "read first")
		mustRunStdin(t, root, "Ports and adapters.\n", "set", "T-01", "scope")
		mustRunStdin(t, root, "Nothing else.\n", "set", "T-01", "out of scope")
		bodyB := bodyBelowH1(t, readText(t, filepath.Join(root, nameB)))

		code, stdout, stderr := runDutyStdin(t, root, bodyB, "create", "task", "One shot", "--body")
		if code != 0 || stderr != "" {
			t.Fatalf("create --body: code=%d stderr=%q", code, stderr)
		}
		nameA := filepath.Base(strings.TrimSuffix(stdout, "\n"))
		bodyA := bodyBelowH1(t, readText(t, filepath.Join(root, nameA)))
		if bodyA != bodyB {
			t.Errorf("one-shot body below H1 =\n%q\nwant (N set calls):\n%q", bodyA, bodyB)
		}
	})

	t.Run("gates authored in the body are counted", func(t *testing.T) {
		root := initDuty(t)
		body := "## Gates\n- [ ] build passes\n- [x] tests green\n\n## Report\n"
		if code, _, stderr := runDutyStdin(t, root, body, "create", "task", "Gated", "--body"); code != 0 || stderr != "" {
			t.Fatalf("create --body: code=%d stderr=%q", code, stderr)
		}
		code, out, stderr := runDuty(t, root, "gates", "T-01", "--agent")
		if code != 0 || stderr != "" {
			t.Fatalf("gates --agent: code=%d stderr=%q", code, stderr)
		}
		if out != "1\tfalse\tbuild passes\n2\ttrue\ttests green\n" {
			t.Errorf("gates --agent =\n%q, want the two body gates", out)
		}
	})

	t.Run("refuses empty and non-heading stdin, writing nothing", func(t *testing.T) {
		for _, tc := range []struct{ name, input string }{
			{"empty", ""},
			{"blank", "\n \t\n"},
			{"non-heading", "just some prose\n"},
		} {
			root := initDuty(t)
			before := readText(t, filepath.Join(root, "BOARD.md"))
			code, stdout, stderr := runDutyStdin(t, root, tc.input, "create", "task", "Bad", "--body")
			if code == 0 {
				t.Fatalf("%s: create --body succeeded, want refusal", tc.name)
			}
			oneLine(t, "stderr", stderr)
			if stdout != "" {
				t.Errorf("%s: stdout = %q, want empty", tc.name, stdout)
			}
			if readText(t, filepath.Join(root, "BOARD.md")) != before {
				t.Errorf("%s: board changed by refused create", tc.name)
			}
			if m, _ := filepath.Glob(filepath.Join(root, "T-*.md")); len(m) != 0 {
				t.Errorf("%s: task file written by refused create: %v", tc.name, m)
			}
		}
	})
}

func TestSetSections(t *testing.T) {
	t.Run("replaces three sections in one call, bytes outside them untouched", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Bulk me")
		before := readText(t, filepath.Join(root, name))
		payload := "## Goal\nG.\n\n## Scope\nS.\n\n## Design\nD.\n"
		mustRunStdin(t, root, payload, "set", "T-01")
		want := replaceOnce(t, before, "## Goal\n\n", "## Goal\nG.\n\n")
		want = replaceOnce(t, want, "## Scope\n\n", "## Scope\nS.\n\n")
		want = replaceOnce(t, want, "## Report\n", "## Design\nD.\n\n## Report\n")
		if got := readText(t, filepath.Join(root, name)); got != want {
			t.Errorf("bulk set =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("single-section form still works", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Still single")
		before := readText(t, filepath.Join(root, name))
		mustRunStdin(t, root, "The stated outcome.\n", "set", "T-01", "goal")
		want := replaceOnce(t, before, "## Goal\n\n", "## Goal\nThe stated outcome.\n\n")
		if got := readText(t, filepath.Join(root, name)); got != want {
			t.Errorf("single set =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("refuses empty and non-heading stdin, changing nothing", func(t *testing.T) {
		for _, input := range []string{"", "\n \t\n", "prose without a heading\n"} {
			root := initDuty(t)
			name := createTask(t, root, "Bulk me")
			before := readText(t, filepath.Join(root, name))
			code, stdout, stderr := runDutyStdin(t, root, input, "set", "T-01")
			if code == 0 {
				t.Fatalf("bulk set with stdin %q succeeded, want refusal", input)
			}
			oneLine(t, "stderr", stderr)
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if readText(t, filepath.Join(root, name)) != before {
				t.Errorf("task file changed by refused bulk set with %q", input)
			}
		}
	})

	t.Run("rejects archived ids", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old-work.md")
		code, _, stderr := runDutyStdin(t, root, "## Goal\nx\n", "set", "T-90")
		if code == 0 {
			t.Fatal("bulk set on an archived id succeeded")
		}
		if !strings.Contains(stderr, "archived") {
			t.Errorf("stderr = %q, want it to say archived", stderr)
		}
	})
}
