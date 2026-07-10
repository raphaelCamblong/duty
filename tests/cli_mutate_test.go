package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raphaelCamblong/duty/internal/cli"
)

// runDutyStdin invokes cli.Run from dir feeding input on stdin and returns
// the exit code plus captured stdout and stderr.
func runDutyStdin(t *testing.T, dir, input string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	t.Chdir(dir)
	var out, errBuf bytes.Buffer
	code = cli.Run(args, strings.NewReader(input), &out, &errBuf, "test")
	return code, out.String(), errBuf.String()
}

// createTask creates a task via the CLI in boardDir and returns its filename.
func createTask(t *testing.T, boardDir, title string) string {
	t.Helper()
	code, stdout, stderr := runDuty(t, boardDir, "create", "task", title)
	if code != 0 {
		t.Fatalf("create %q: code=%d stderr=%q", title, code, stderr)
	}
	return filepath.Base(strings.TrimSuffix(stdout, "\n"))
}

// mustRun runs a duty command that has to succeed quietly.
func mustRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	code, stdout, stderr := runDuty(t, dir, args...)
	if code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("duty %v: code=%d stdout=%q stderr=%q", args, code, stdout, stderr)
	}
}

// writeArchived plants an archived task file under root/archive.
func writeArchived(t *testing.T, root, filename string) {
	t.Helper()
	path := filepath.Join(root, "archive", filename)
	if err := os.WriteFile(path, []byte("---\nid: T-90\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStatus(t *testing.T) {
	t.Run("rewrites frontmatter and board cell in one call", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "First task")
		mustRun(t, root, "status", "T-01", "in-progress")
		file := readText(t, filepath.Join(root, name))
		if !strings.Contains(file, "\nstatus: in-progress\n") {
			t.Errorf("task file status not updated: %q", file)
		}
		wantRow := "| [T-01](" + name + ") | First task | in-progress |"
		if got := readText(t, filepath.Join(root, "BOARD.md")); !strings.Contains(got, wantRow) {
			t.Errorf("board missing row %q in %q", wantRow, got)
		}
	})

	t.Run("every known status is accepted", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Cycle me")
		for _, status := range []string{"in-progress", "blocked", "done", "todo"} {
			mustRun(t, root, "status", "T-01", status)
			if got := readText(t, filepath.Join(root, name)); !strings.Contains(got, "status: "+status+"\n") {
				t.Errorf("status %s: file has %q", status, got)
			}
		}
	})

	t.Run("rejects unknown status and writes nothing", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "First task")
		taskBefore := readText(t, filepath.Join(root, name))
		boardBefore := readText(t, filepath.Join(root, "BOARD.md"))
		code, stdout, stderr := runDuty(t, root, "status", "T-01", "cancelled")
		if code == 0 {
			t.Fatal("unknown status accepted")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "cancelled") {
			t.Errorf("stderr = %q, want it to name the bad status", stderr)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if readText(t, filepath.Join(root, name)) != taskBefore {
			t.Error("task file changed by rejected status")
		}
		if readText(t, filepath.Join(root, "BOARD.md")) != boardBefore {
			t.Error("board changed by rejected status")
		}
	})

	t.Run("rejects archived ids", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old-work.md")
		code, _, stderr := runDuty(t, root, "status", "T-90", "todo")
		if code == 0 {
			t.Fatal("status on an archived id succeeded")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "archived") {
			t.Errorf("stderr = %q, want it to say archived", stderr)
		}
	})

	t.Run("unknown id errors", func(t *testing.T) {
		root := initDuty(t)
		code, _, stderr := runDuty(t, root, "status", "T-99", "todo")
		if code == 0 {
			t.Fatal("status on an unknown id succeeded")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "T-99") {
			t.Errorf("stderr = %q, want it to name T-99", stderr)
		}
	})

	t.Run("resolves ids anywhere in the tree", func(t *testing.T) {
		root := initDuty(t)
		mustRun(t, root, "create", "track", "backend")
		sub := filepath.Join(root, "backend")
		name := createTask(t, sub, "Backend task")
		mustRun(t, root, "status", "T-01", "done")
		if got := readText(t, filepath.Join(sub, name)); !strings.Contains(got, "status: done\n") {
			t.Errorf("sub-board task file not updated: %q", got)
		}
		if got := readText(t, filepath.Join(sub, "BOARD.md")); !strings.Contains(got, "| done |") {
			t.Errorf("sub-board row not updated: %q", got)
		}
	})

	t.Run("argument validation", func(t *testing.T) {
		for _, args := range [][]string{
			{"status"},
			{"status", "T-01"},
			{"status", "T-01", "todo", "extra"},
		} {
			root := initDuty(t)
			createTask(t, root, "First task")
			code, _, stderr := runDuty(t, root, args...)
			if code == 0 {
				t.Errorf("duty %v succeeded, want usage error", args)
			}
			oneLine(t, "stderr", stderr)
		}
	})
}

func TestMoveSection(t *testing.T) {
	t.Run("moves the row under a new section above the footer", func(t *testing.T) {
		root := initDuty(t)
		one := createTask(t, root, "First task")
		two := createTask(t, root, "Second task")
		mustRun(t, root, "move", "T-01", "--section", "Later")
		got := readText(t, filepath.Join(root, "BOARD.md"))
		openAt := strings.Index(got, "\n## Open tasks\n")
		twoAt := strings.Index(got, "("+two+")")
		laterAt := strings.Index(got, "\n## Later\n")
		oneAt := strings.Index(got, "("+one+")")
		footerAt := strings.Index(got, "Completed tasks (0)")
		if openAt < 0 || twoAt < openAt || laterAt < twoAt || oneAt < laterAt || footerAt < oneAt {
			t.Errorf("want Open tasks(T-02) then Later(T-01) then footer, got %q", got)
		}
		if strings.Count(got, "("+one+")") != 1 {
			t.Errorf("row for %s duplicated in %q", one, got)
		}
	})

	t.Run("appends to an existing section", func(t *testing.T) {
		root := initDuty(t)
		one := createTask(t, root, "First task")
		two := createTask(t, root, "Second task")
		mustRun(t, root, "move", "T-01", "--section", "Later")
		mustRun(t, root, "move", "T-02", "--section", "Later")
		got := readText(t, filepath.Join(root, "BOARD.md"))
		if strings.Count(got, "## Later") != 1 {
			t.Fatalf("want one Later section, got %q", got)
		}
		laterAt := strings.Index(got, "## Later")
		oneAt := strings.Index(got, "("+one+")")
		twoAt := strings.Index(got, "("+two+")")
		if oneAt < laterAt || twoAt < oneAt {
			t.Errorf("want both rows under Later in move order, got %q", got)
		}
	})

	t.Run("prunes the section it empties", func(t *testing.T) {
		root := initDuty(t)
		one := createTask(t, root, "First task")
		mustRun(t, root, "move", "T-01", "--section", "Later")
		mustRun(t, root, "move", "T-01", "--section", "Open tasks")
		got := readText(t, filepath.Join(root, "BOARD.md"))
		if strings.Contains(got, "## Later") {
			t.Errorf("emptied section not pruned: %q", got)
		}
		openAt := strings.Index(got, "## Open tasks")
		oneAt := strings.Index(got, "("+one+")")
		if openAt < 0 || oneAt < openAt {
			t.Errorf("row not back under Open tasks: %q", got)
		}
	})

	t.Run("never prunes the default section", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Only task")
		mustRun(t, root, "move", "T-01", "--section", "Later")
		if got := readText(t, filepath.Join(root, "BOARD.md")); !strings.Contains(got, "## Open tasks") {
			t.Errorf("default section pruned: %q", got)
		}
	})

	t.Run("rejects archived ids", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old-work.md")
		code, _, stderr := runDuty(t, root, "move", "T-90", "--section", "Later")
		if code == 0 {
			t.Fatal("move --section on an archived id succeeded")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "archived") {
			t.Errorf("stderr = %q, want it to say archived", stderr)
		}
	})

	t.Run("errors", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{name: "unknown id", args: []string{"move", "T-99", "--section", "Later"}},
			{name: "no flags", args: []string{"move", "T-01"}},
			{name: "no args", args: []string{"move"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				root := initDuty(t)
				createTask(t, root, "First task")
				before := readText(t, filepath.Join(root, "BOARD.md"))
				code, stdout, stderr := runDuty(t, root, tt.args...)
				if code == 0 {
					t.Fatalf("duty %v succeeded, want error", tt.args)
				}
				oneLine(t, "stderr", stderr)
				if stdout != "" {
					t.Errorf("stdout = %q, want empty", stdout)
				}
				if readText(t, filepath.Join(root, "BOARD.md")) != before {
					t.Error("board changed by failed move")
				}
			})
		}
	})
}

func TestReport(t *testing.T) {
	t.Run("appends stdin under the Report heading", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "First task")
		code, stdout, stderr := runDutyStdin(t, root, "Did the thing.\n", "report", "T-01")
		if code != 0 || stdout != "" || stderr != "" {
			t.Fatalf("report: code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
		got := readText(t, filepath.Join(root, name))
		if !strings.HasSuffix(got, "## Report\n\nDid the thing.\n") {
			t.Errorf("task file ends %q, want the report under ## Report", got)
		}
	})

	t.Run("reports accumulate", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "First task")
		runDutyStdin(t, root, "First pass.\n", "report", "T-01")
		if code, _, stderr := runDutyStdin(t, root, "Second pass.\n", "report", "T-01"); code != 0 {
			t.Fatalf("second report: %q", stderr)
		}
		got := readText(t, filepath.Join(root, name))
		firstAt := strings.Index(got, "First pass.")
		secondAt := strings.Index(got, "Second pass.")
		if firstAt < 0 || secondAt < firstAt {
			t.Errorf("want both reports in order, got %q", got)
		}
		if strings.Count(got, "## Report") != 1 {
			t.Errorf("Report heading duplicated: %q", got)
		}
	})

	t.Run("refuses empty stdin", func(t *testing.T) {
		for _, input := range []string{"", "\n \t\n"} {
			root := initDuty(t)
			name := createTask(t, root, "First task")
			before := readText(t, filepath.Join(root, name))
			code, stdout, stderr := runDutyStdin(t, root, input, "report", "T-01")
			if code == 0 {
				t.Fatalf("report with stdin %q succeeded, want refusal", input)
			}
			oneLine(t, "stderr", stderr)
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if readText(t, filepath.Join(root, name)) != before {
				t.Error("task file changed by refused report")
			}
		}
	})

	t.Run("rejects archived ids", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old-work.md")
		code, _, stderr := runDutyStdin(t, root, "Late words.\n", "report", "T-90")
		if code == 0 {
			t.Fatal("report on an archived id succeeded")
		}
		oneLine(t, "stderr", stderr)
	})

	t.Run("unknown id errors", func(t *testing.T) {
		root := initDuty(t)
		code, _, stderr := runDutyStdin(t, root, "Words.\n", "report", "T-99")
		if code == 0 {
			t.Fatal("report on an unknown id succeeded")
		}
		oneLine(t, "stderr", stderr)
	})
}

func TestMoveTrack(t *testing.T) {
	t.Run("renames the file and swaps rows, preserving status", func(t *testing.T) {
		root := initDuty(t)
		mustRun(t, root, "create", "track", "backend")
		name := createTask(t, root, "Move me")
		mustRun(t, root, "status", "T-01", "in-progress")
		mustRun(t, root, "move", "T-01", "--track", "backend")

		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("source file still present (err %v)", err)
		}
		moved := filepath.Join(root, "backend", name)
		if got := readText(t, moved); !strings.Contains(got, "status: in-progress\n") {
			t.Errorf("moved file lost its status: %q", got)
		}
		if got := readText(t, filepath.Join(root, "BOARD.md")); strings.Contains(got, "("+name+")") {
			t.Errorf("source board still has the row: %q", got)
		}
		got := readText(t, filepath.Join(root, "backend", "BOARD.md"))
		wantRow := "| [T-01](" + name + ") | Move me | in-progress |"
		openAt := strings.Index(got, "## Open tasks")
		rowAt := strings.Index(got, wantRow)
		footerAt := strings.Index(got, "Completed tasks (0)")
		if openAt < 0 || rowAt < openAt || footerAt < rowAt {
			t.Errorf("target board missing row %q under Open tasks: %q", wantRow, got)
		}
	})

	t.Run("move there and back restores both boards and the file", func(t *testing.T) {
		root := initDuty(t)
		mustRun(t, root, "create", "track", "backend")
		name := createTask(t, root, "Round trip")
		rootBoard := readText(t, filepath.Join(root, "BOARD.md"))
		subBoard := readText(t, filepath.Join(root, "backend", "BOARD.md"))
		file := readText(t, filepath.Join(root, name))

		mustRun(t, root, "move", "T-01", "--track", "backend")
		mustRun(t, filepath.Join(root, "backend"), "move", "T-01", "--track", ".")

		if got := readText(t, filepath.Join(root, "BOARD.md")); got != rootBoard {
			t.Errorf("root board not restored:\n got %q\nwant %q", got, rootBoard)
		}
		if got := readText(t, filepath.Join(root, "backend", "BOARD.md")); got != subBoard {
			t.Errorf("sub board not restored:\n got %q\nwant %q", got, subBoard)
		}
		if got := readText(t, filepath.Join(root, name)); got != file {
			t.Errorf("task file not restored:\n got %q\nwant %q", got, file)
		}
	})

	t.Run("section flag places the row and prunes the source section", func(t *testing.T) {
		root := initDuty(t)
		mustRun(t, root, "create", "track", "backend")
		name := createTask(t, root, "Sectioned")
		mustRun(t, root, "move", "T-01", "--section", "Later")
		mustRun(t, root, "move", "T-01", "--track", "backend", "--section", "Doing")

		src := readText(t, filepath.Join(root, "BOARD.md"))
		if strings.Contains(src, "## Later") {
			t.Errorf("emptied source section not pruned: %q", src)
		}
		got := readText(t, filepath.Join(root, "backend", "BOARD.md"))
		doingAt := strings.Index(got, "\n## Doing\n")
		rowAt := strings.Index(got, "("+name+")")
		footerAt := strings.Index(got, "Completed tasks (0)")
		if doingAt < 0 || rowAt < doingAt || footerAt < rowAt {
			t.Errorf("want row under ## Doing above the footer, got %q", got)
		}
	})

	t.Run("moves into nested boards by path", func(t *testing.T) {
		root := initDuty(t)
		mustRun(t, root, "create", "track", "backend")
		mustRun(t, filepath.Join(root, "backend"), "create", "track", "api")
		name := createTask(t, root, "Deep task")
		mustRun(t, root, "move", "T-01", "--track", "backend/api")
		if _, err := os.Stat(filepath.Join(root, "backend", "api", name)); err != nil {
			t.Errorf("file not in nested board: %v", err)
		}
		if !strings.Contains(readText(t, filepath.Join(root, "backend", "api", "BOARD.md")), "("+name+")") {
			t.Error("nested board missing the row")
		}
	})

	t.Run("move to the same board keeps one row and the file", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Stay put")
		mustRun(t, root, "move", "T-01", "--track", ".")
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("file missing after same-board move: %v", err)
		}
		if got := readText(t, filepath.Join(root, "BOARD.md")); strings.Count(got, "("+name+")") != 1 {
			t.Errorf("want exactly one row, got %q", got)
		}
	})

	t.Run("errors leave everything untouched", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{name: "non-existent track", args: []string{"move", "T-01", "--track", "nope"}},
			{name: "path escaping the tree", args: []string{"move", "T-01", "--track", "../elsewhere"}},
			{name: "absolute path", args: []string{"move", "T-01", "--track", "/tmp"}},
			{name: "unknown id", args: []string{"move", "T-99", "--track", "."}},
			{name: "old positional spelling", args: []string{"move", "T-01", "backend"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				root := initDuty(t)
				name := createTask(t, root, "First task")
				boardBefore := readText(t, filepath.Join(root, "BOARD.md"))
				code, stdout, stderr := runDuty(t, root, tt.args...)
				if code == 0 {
					t.Fatalf("duty %v succeeded, want error", tt.args)
				}
				oneLine(t, "stderr", stderr)
				if stdout != "" {
					t.Errorf("stdout = %q, want empty", stdout)
				}
				if _, err := os.Stat(filepath.Join(root, name)); err != nil {
					t.Errorf("task file moved by failed move: %v", err)
				}
				if readText(t, filepath.Join(root, "BOARD.md")) != boardBefore {
					t.Error("board changed by failed move")
				}
			})
		}
	})

	t.Run("rejects archived ids", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old-work.md")
		code, _, stderr := runDuty(t, root, "move", "T-90", "--track", ".")
		if code == 0 {
			t.Fatal("move on an archived id succeeded")
		}
		oneLine(t, "stderr", stderr)
	})
}
