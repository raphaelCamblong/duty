package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/cli"
	"github.com/raphaelCamblong/duty/internal/task"
)

// runDuty invokes cli.Run from dir with empty stdin and returns the exit code
// plus captured stdout and stderr.
func runDuty(t *testing.T, dir string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	t.Chdir(dir)
	var out, errBuf bytes.Buffer
	code = cli.Run(args, strings.NewReader(""), &out, &errBuf, "test")
	return code, out.String(), errBuf.String()
}

// initDuty bootstraps a fresh duty tree in a t.TempDir() via the init command
// and returns the tree root (the duty/ directory).
func initDuty(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	code, stdout, stderr := runDuty(t, dir, "init")
	if code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("init: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	return filepath.Join(dir, "duty")
}

// readText reads path as a string, failing the test on error.
func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// samePath reports whether two paths name the same file once symlinks are
// resolved (macOS temp dirs live behind a /var -> /private/var symlink).
func samePath(t *testing.T, a, b string) bool {
	t.Helper()
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		t.Fatalf("eval %s: %v", a, err)
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		t.Fatalf("eval %s: %v", b, err)
	}
	return ra == rb
}

// oneLine asserts that s is exactly one non-empty \n-terminated line.
func oneLine(t *testing.T, name, s string) {
	t.Helper()
	if s == "" || !strings.HasSuffix(s, "\n") || strings.Count(s, "\n") != 1 {
		t.Errorf("%s: want exactly one line, got %q", name, s)
	}
}

// reportHeadingRE matches a report append's dated heading line: "### 2006-01-02
// 15:04", plus " — status" when a status was given.
var reportHeadingRE = regexp.MustCompile(`(?m)^### \d{4}-\d{2}-\d{2} \d{2}:\d{2}(?: — \S+)?$`)

// reportHeadingIn returns the single dated report heading found in content,
// failing the test when none or more than one is present — the seam
// real-clock CLI tests use to splice the actual stamp into a byte-exact
// fixture instead of racing it.
func reportHeadingIn(t *testing.T, content string) string {
	t.Helper()
	matches := reportHeadingRE.FindAllString(content, -1)
	if len(matches) != 1 {
		t.Fatalf("content has %d dated report headings, want exactly 1:\n%q", len(matches), content)
	}
	return matches[0]
}

func TestRunDispatch(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantErr  string
	}{
		{name: "no command", args: nil, wantCode: 2, wantErr: "usage: duty <command> [args]\n"},
		{name: "unknown command", args: []string{"zzz"}, wantCode: 2, wantErr: "unknown command \"zzz\"\n"},
		{name: "typo suggests the close command", args: []string{"creat"}, wantCode: 2, wantErr: "unknown command \"creat\" — did you mean \"create\"?\n"},
		{name: "old create spelling", args: []string{"create", "First task"}, wantCode: 2, wantErr: "unknown command \"First task\"\n"},
		{name: "removed track command", args: []string{"track", "backend"}, wantCode: 2, wantErr: "unknown command \"track\"\n"},
		{name: "removed board command", args: []string{"board", "backend"}, wantCode: 2, wantErr: "unknown command \"board\"\n"},
		{name: "removed link command", args: []string{"link", "T-01", "Later"}, wantCode: 2, wantErr: "unknown command \"link\"\n"},
		{name: "old delete spelling", args: []string{"delete", "T-01"}, wantCode: 2, wantErr: "unknown command \"T-01\"\n"},
		{name: "bare create", args: []string{"create"}, wantCode: 2, wantErr: "usage: duty create <task|track> [args]\n"},
		{name: "bare get", args: []string{"get"}, wantCode: 2, wantErr: "usage: duty get <task|tasks|tracks|next> [args]\n"},
		{name: "bare delete", args: []string{"delete"}, wantCode: 2, wantErr: "usage: duty delete task <id> [--force]\n"},
		{name: "unknown get resource", args: []string{"get", "nope"}, wantCode: 2, wantErr: "unknown command \"nope\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runDuty(t, t.TempDir(), tt.args...)
			if code != tt.wantCode {
				t.Errorf("code = %d, want %d", code, tt.wantCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr != tt.wantErr {
				t.Errorf("stderr = %q, want %q", stderr, tt.wantErr)
			}
		})
	}
}

func TestInit(t *testing.T) {
	t.Run("bootstraps the skeleton tree quietly", func(t *testing.T) {
		root := initDuty(t)
		if got, want := readText(t, filepath.Join(root, "BOARD.md")), string(board.Render("Board")); got != want {
			t.Errorf("BOARD.md = %q, want %q", got, want)
		}
		readme := readText(t, filepath.Join(root, "README.md"))
		for _, want := range []string{"# duty/", "duty status <id> <status>", "## Lifecycle → command"} {
			if !strings.Contains(readme, want) {
				t.Errorf("README.md missing %q", want)
			}
		}
		info, err := os.Stat(filepath.Join(root, "archive"))
		if err != nil || !info.IsDir() {
			t.Errorf("archive/ not a directory: %v", err)
		}
	})

	t.Run("title argument becomes the H1", func(t *testing.T) {
		dir := t.TempDir()
		if code, _, stderr := runDuty(t, dir, "init", "My Project"); code != 0 {
			t.Fatalf("init: code=%d stderr=%q", code, stderr)
		}
		got := readText(t, filepath.Join(dir, "duty", "BOARD.md"))
		if !strings.HasPrefix(got, "# My Project\n") {
			t.Errorf("BOARD.md starts %q, want H1 %q", got[:min(len(got), 20)], "# My Project")
		}
	})

	t.Run("refuses re-init next to the tree", func(t *testing.T) {
		root := initDuty(t)
		before := readText(t, filepath.Join(root, "BOARD.md"))
		code, stdout, stderr := runDuty(t, filepath.Dir(root), "init")
		if code == 0 {
			t.Fatal("re-init succeeded, want refusal")
		}
		oneLine(t, "stderr", stderr)
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if got := readText(t, filepath.Join(root, "BOARD.md")); got != before {
			t.Error("BOARD.md changed by refused init")
		}
	})

	t.Run("refuses inside a board directory", func(t *testing.T) {
		root := initDuty(t)
		if code, _, stderr := runDuty(t, root, "init"); code == 0 || stderr == "" {
			t.Errorf("init inside board: code=%d stderr=%q, want refusal", code, stderr)
		}
	})

	t.Run("rejects extra arguments", func(t *testing.T) {
		code, _, stderr := runDuty(t, t.TempDir(), "init", "one", "two")
		if code == 0 {
			t.Fatal("init with two args succeeded")
		}
		oneLine(t, "stderr", stderr)
	})
}

func TestCreate(t *testing.T) {
	t.Run("writes task file and board row in one call", func(t *testing.T) {
		root := initDuty(t)
		code, stdout, stderr := runDuty(t, root, "create", "task", "First task")
		if code != 0 || stderr != "" {
			t.Fatalf("create: code=%d stderr=%q", code, stderr)
		}
		oneLine(t, "stdout", stdout)
		id, printed := splitCreateOutput(t, stdout)
		if id != "T-01" {
			t.Errorf("printed id %q, want T-01", id)
		}
		taskPath := filepath.Join(root, "T-01-first-task.md")
		if !samePath(t, printed, taskPath) {
			t.Errorf("printed path %q, want %q", printed, taskPath)
		}
		if got, want := readText(t, taskPath), string(task.Render("T-01", "First task", nil)); got != want {
			t.Errorf("task file = %q, want %q", got, want)
		}
		wantRow := "| [T-01](T-01-first-task.md) | First task | todo |"
		if got := readText(t, filepath.Join(root, "BOARD.md")); !strings.Contains(got, wantRow) {
			t.Errorf("board missing row %q in %q", wantRow, got)
		}
	})

	t.Run("accepts flags after the title", func(t *testing.T) {
		root := initDuty(t)
		code, stdout, stderr := runDuty(t, root, "create", "task", "Second task", "--slug", "second")
		if code != 0 {
			t.Fatalf("create: code=%d stderr=%q", code, stderr)
		}
		if want := "T-01-second.md"; !strings.Contains(stdout, want) {
			t.Errorf("stdout = %q, want it to name %s", stdout, want)
		}
		if _, err := os.Stat(filepath.Join(root, "T-01-second.md")); err != nil {
			t.Errorf("slugged file missing: %v", err)
		}
	})

	t.Run("numbers tree-wide including archive", func(t *testing.T) {
		root := initDuty(t)
		archived := filepath.Join(root, "archive", "T-07-old.md")
		if err := os.WriteFile(archived, []byte("---\nid: T-07\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		code, stdout, stderr := runDuty(t, root, "create", "task", "Next task")
		if code != 0 {
			t.Fatalf("create: code=%d stderr=%q", code, stderr)
		}
		if want := "T-08-next-task.md"; !strings.Contains(stdout, want) {
			t.Errorf("stdout = %q, want %s (archived T-07 blocks reuse)", stdout, want)
		}
	})

	t.Run("blocked-by resolves anywhere in the tree", func(t *testing.T) {
		root := initDuty(t)
		if code, _, stderr := runDuty(t, root, "create", "task", "Root task"); code != 0 {
			t.Fatalf("create root task: %q", stderr)
		}
		if code, _, stderr := runDuty(t, root, "create", "track", "backend"); code != 0 {
			t.Fatalf("board: %q", stderr)
		}
		sub := filepath.Join(root, "backend")
		code, _, stderr := runDuty(t, sub, "create", "task", "Backend task", "--blocked-by", "T-01")
		if code != 0 {
			t.Fatalf("create in sub-board: code=%d stderr=%q", code, stderr)
		}
		got := readText(t, filepath.Join(sub, "T-02-backend-task.md"))
		if !strings.Contains(got, "blocked-by: [T-01]") {
			t.Errorf("frontmatter missing blocked-by, got %q", got)
		}
		if !strings.Contains(readText(t, filepath.Join(sub, "BOARD.md")), "(T-02-backend-task.md)") {
			t.Error("sub-board missing the new row")
		}
	})

	t.Run("rejects unknown blocked-by and writes nothing", func(t *testing.T) {
		root := initDuty(t)
		boardPath := filepath.Join(root, "BOARD.md")
		before := readText(t, boardPath)
		code, stdout, stderr := runDuty(t, root, "create", "task", "Bad deps", "--blocked-by", "T-99")
		if code == 0 {
			t.Fatal("create with unknown blocked-by succeeded")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "T-99") {
			t.Errorf("stderr = %q, want it to name T-99", stderr)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if got := readText(t, boardPath); got != before {
			t.Error("board changed by refused create")
		}
		matches, err := filepath.Glob(filepath.Join(root, "T-*.md"))
		if err != nil || len(matches) != 0 {
			t.Errorf("task files after refused create: %v (err %v)", matches, err)
		}
	})

	t.Run("accepts an archived blocked-by id", func(t *testing.T) {
		root := initDuty(t)
		archived := filepath.Join(root, "archive", "T-03-done-work.md")
		if err := os.WriteFile(archived, []byte("---\nid: T-03\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		code, _, stderr := runDuty(t, root, "create", "task", "Follow up", "--blocked-by", "T-03")
		if code != 0 {
			t.Errorf("create: code=%d stderr=%q, want success (archived dep exists)", code, stderr)
		}
	})

	t.Run("section flag creates the section", func(t *testing.T) {
		root := initDuty(t)
		code, _, stderr := runDuty(t, root, "create", "task", "Later task", "--section", "Later")
		if code != 0 {
			t.Fatalf("create: code=%d stderr=%q", code, stderr)
		}
		got := readText(t, filepath.Join(root, "BOARD.md"))
		sectionAt := strings.Index(got, "\n## Later\n")
		rowAt := strings.Index(got, "| [T-01](T-01-later-task.md) | Later task | todo |")
		footerAt := strings.Index(got, "Completed tasks (0)")
		if sectionAt < 0 || rowAt < sectionAt || footerAt < rowAt {
			t.Errorf("want section, then row, then footer; got %q", got)
		}
	})

	t.Run("argument and flag validation", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{name: "missing title", args: []string{"create", "task"}},
			{name: "two titles", args: []string{"create", "task", "one", "two"}},
			{name: "invalid slug", args: []string{"create", "task", "Task", "--slug", "Bad_Slug"}},
			{name: "unslugifiable title", args: []string{"create", "task", "!!!"}},
			{name: "unknown flag", args: []string{"create", "task", "Task", "--nope"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				root := initDuty(t)
				code, stdout, stderr := runDuty(t, root, tt.args...)
				if code == 0 {
					t.Fatal("create succeeded, want error")
				}
				oneLine(t, "stderr", stderr)
				if stdout != "" {
					t.Errorf("stdout = %q, want empty", stdout)
				}
			})
		}
	})
}

func TestCreateTrack(t *testing.T) {
	t.Run("creates skeleton and parent bullet quietly", func(t *testing.T) {
		root := initDuty(t)
		code, stdout, stderr := runDuty(t, root, "create", "track", "backend", "--title", "Backend")
		if code != 0 || stdout != "" || stderr != "" {
			t.Fatalf("create track: code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
		sub := filepath.Join(root, "backend")
		if got, want := readText(t, filepath.Join(sub, "BOARD.md")), string(board.Render("Backend")); got != want {
			t.Errorf("sub BOARD.md = %q, want %q", got, want)
		}
		info, err := os.Stat(filepath.Join(sub, "archive"))
		if err != nil || !info.IsDir() {
			t.Errorf("backend/archive not a directory: %v", err)
		}
		parent := readText(t, filepath.Join(root, "BOARD.md"))
		if !strings.Contains(parent, "## Boards") {
			t.Error("parent missing ## Boards section")
		}
		if want := "- [backend/](backend/BOARD.md) — Backend"; !strings.Contains(parent, want) {
			t.Errorf("parent missing bullet %q in %q", want, parent)
		}
	})

	t.Run("title defaults to the name", func(t *testing.T) {
		root := initDuty(t)
		if code, _, stderr := runDuty(t, root, "create", "track", "api"); code != 0 {
			t.Fatalf("create track: %q", stderr)
		}
		if got := readText(t, filepath.Join(root, "api", "BOARD.md")); !strings.HasPrefix(got, "# api\n") {
			t.Errorf("sub H1 = %q, want # api", got[:min(len(got), 10)])
		}
		if want := "- [api/](api/BOARD.md) — api"; !strings.Contains(readText(t, filepath.Join(root, "BOARD.md")), want) {
			t.Errorf("parent missing bullet %q", want)
		}
	})

	t.Run("nests under the current board", func(t *testing.T) {
		root := initDuty(t)
		if code, _, stderr := runDuty(t, root, "create", "track", "backend"); code != 0 {
			t.Fatalf("create track backend: %q", stderr)
		}
		sub := filepath.Join(root, "backend")
		if code, _, stderr := runDuty(t, sub, "create", "track", "api"); code != 0 {
			t.Fatalf("create track api: %q", stderr)
		}
		if _, err := os.Stat(filepath.Join(sub, "api", "BOARD.md")); err != nil {
			t.Errorf("nested board missing: %v", err)
		}
		if want := "- [api/](api/BOARD.md) — api"; !strings.Contains(readText(t, filepath.Join(sub, "BOARD.md")), want) {
			t.Error("backend board missing the api bullet")
		}
	})

	t.Run("refuses an existing folder", func(t *testing.T) {
		root := initDuty(t)
		if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
			t.Fatal(err)
		}
		before := readText(t, filepath.Join(root, "BOARD.md"))
		code, _, stderr := runDuty(t, root, "create", "track", "docs")
		if code == 0 {
			t.Fatal("create track over an existing folder succeeded")
		}
		oneLine(t, "stderr", stderr)
		if got := readText(t, filepath.Join(root, "BOARD.md")); got != before {
			t.Error("parent board changed by refused create track")
		}
	})

	t.Run("name validation", func(t *testing.T) {
		for _, name := range []string{"Backend", "back end", "back/end", "", "café"} {
			t.Run(name, func(t *testing.T) {
				root := initDuty(t)
				code, _, stderr := runDuty(t, root, "create", "track", name)
				if code == 0 {
					t.Fatalf("create track %q succeeded, want invalid-name error", name)
				}
				oneLine(t, "stderr", stderr)
			})
		}
	})
}
