package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raphaelCamblong/duty/internal/board"
)

// pathValue extracts the value of the "path:" line from get task's human
// output.
func pathValue(t *testing.T, stdout string) string {
	t.Helper()
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "path:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "path:"))
		}
	}
	t.Fatalf("no path: line in %q", stdout)
	return ""
}

func TestGetTask(t *testing.T) {
	t.Run("human form shows aligned metadata and the file path", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "First task")
		mustRun(t, root, "status", "T-01", "in-progress")

		code, stdout, stderr := runDuty(t, root, "get", "task", "T-01")
		if code != 0 || stderr != "" {
			t.Fatalf("get task: code=%d stderr=%q", code, stderr)
		}
		for _, want := range []string{"T-01", "First task", "in-progress", "none", "0/0"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("stdout missing %q:\n%s", want, stdout)
			}
		}
		last := -1
		for _, key := range []string{"id:", "title:", "status:", "track:", "blocked-by:", "gates:", "path:"} {
			at := strings.Index(stdout, key)
			if at <= last {
				t.Errorf("key %q out of order in %q", key, stdout)
			}
			last = at
		}
		if got := pathValue(t, stdout); !samePath(t, got, filepath.Join(root, name)) {
			t.Errorf("path = %q, want %s", got, filepath.Join(root, name))
		}
	})

	t.Run("resolves a sub-track task and lists its blocked-by", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Dep")
		mustRun(t, root, "create", "track", "backend")
		sub := filepath.Join(root, "backend")
		if code, _, stderr := runDuty(t, sub, "create", "task", "Blocked task", "--blocked-by", "T-01"); code != 0 {
			t.Fatalf("create: code=%d stderr=%q", code, stderr)
		}
		code, stdout, stderr := runDuty(t, sub, "get", "task", "T-02")
		if code != 0 || stderr != "" {
			t.Fatalf("get task: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "track:") || !strings.Contains(stdout, "backend") {
			t.Errorf("stdout = %q, want track backend", stdout)
		}
		if !strings.Contains(stdout, "blocked-by: T-01") {
			t.Errorf("stdout = %q, want blocked-by T-01", stdout)
		}
	})

	t.Run("counts gate checkboxes", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Gated")
		content := "---\nid: T-01\ntitle: Gated\nstatus: todo\nblocked-by: []\n---\n\n## Gates\n\n- [x] one\n- [ ] two\n- [ ] three\n"
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		_, stdout, _ := runDuty(t, root, "get", "task", "T-01")
		if !strings.Contains(stdout, "1/3") {
			t.Errorf("stdout = %q, want gates 1/3", stdout)
		}
	})

	t.Run("--agent emits the exact TSV field order", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Dep")
		code, createOut, stderr := runDuty(t, root, "create", "task", "Main task", "--blocked-by", "T-01")
		if code != 0 {
			t.Fatalf("create: code=%d stderr=%q", code, stderr)
		}
		mainPath := strings.TrimSpace(createOut)
		mustRun(t, root, "status", "T-02", "in-progress")

		code, stdout, stderr := runDuty(t, root, "get", "task", "T-02", "--agent")
		if code != 0 || stderr != "" {
			t.Fatalf("get task --agent: code=%d stderr=%q", code, stderr)
		}
		oneLine(t, "stdout", stdout)
		fields := strings.Split(strings.TrimRight(stdout, "\n"), "\t")
		if len(fields) != 9 {
			t.Fatalf("record %q: got %d fields, want 9", stdout, len(fields))
		}
		if want := []string{"T-02", ".", "in-progress", "Main task", "0", "0", "T-01"}; !equalFields(fields[:7], want) {
			t.Errorf("fields[:7] = %v, want %v", fields[:7], want)
		}
		if !samePath(t, fields[7], mainPath) {
			t.Errorf("path field = %q, want %s", fields[7], mainPath)
		}
	})

	t.Run("unknown id fails with one stderr line and no output", func(t *testing.T) {
		root := initDuty(t)
		code, stdout, stderr := runDuty(t, root, "get", "task", "T-99")
		if code == 0 {
			t.Fatal("get task on an unknown id succeeded")
		}
		oneLine(t, "stderr", stderr)
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if want := "unknown task id \"T-99\" — try 'duty get tasks'\n"; stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	})

	t.Run("missing id fails", func(t *testing.T) {
		root := initDuty(t)
		code, _, stderr := runDuty(t, root, "get", "task")
		if code == 0 {
			t.Fatal("get task with no id succeeded")
		}
		oneLine(t, "stderr", stderr)
	})
}

func TestGetTracks(t *testing.T) {
	build := func(t *testing.T) string {
		t.Helper()
		root := initDuty(t)
		createTask(t, root, "One")
		createTask(t, root, "Two")
		mustRun(t, root, "status", "T-02", "in-progress")
		createTask(t, root, "Three")
		mustRun(t, root, "status", "T-03", "blocked")
		mustRun(t, root, "create", "track", "backend", "--title", "Backend")
		sub := filepath.Join(root, "backend")
		createTask(t, sub, "B one")
		createTask(t, sub, "B two")
		mustRun(t, sub, "status", "T-05", "done")
		writeArchived(t, root, "T-06-archived.md")
		if err := os.WriteFile(filepath.Join(sub, "archive", "T-07-archived.md"), []byte("---\nid: T-07\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return root
	}

	t.Run("--agent emits one fixed-column record per board, root as \".\"", func(t *testing.T) {
		root := build(t)
		code, stdout, stderr := runDuty(t, root, "get", "tracks", "--agent")
		if code != 0 || stderr != "" {
			t.Fatalf("get tracks --agent: code=%d stderr=%q", code, stderr)
		}
		records := make(map[string][]string)
		for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) != 7 {
				t.Fatalf("record %q: got %d fields, want 7", line, len(fields))
			}
			records[fields[0]] = fields
		}
		if want := []string{".", "Board", "1", "1", "0", "1", "1"}; !equalFields(records["."], want) {
			t.Errorf("root record = %v, want %v", records["."], want)
		}
		if want := []string{"backend", "Backend", "1", "0", "1", "0", "1"}; !equalFields(records["backend"], want) {
			t.Errorf("backend record = %v, want %v", records["backend"], want)
		}
	})

	t.Run("human form shows the counts per track", func(t *testing.T) {
		root := build(t)
		code, stdout, stderr := runDuty(t, root, "get", "tracks")
		if code != 0 || stderr != "" {
			t.Fatalf("get tracks: code=%d stderr=%q", code, stderr)
		}
		for _, want := range []string{
			"1 todo · 1 in-progress · 0 done · 1 blocked · 1 archived",
			"1 todo · 0 in-progress · 1 done · 0 blocked · 1 archived",
			"Board", "Backend",
		} {
			if !strings.Contains(stdout, want) {
				t.Errorf("stdout missing %q:\n%s", want, stdout)
			}
		}
	})

	t.Run("scoped to the current board, which reads as \".\"", func(t *testing.T) {
		root := initDuty(t)
		mustRun(t, root, "create", "track", "backend")
		sub := filepath.Join(root, "backend")
		createTask(t, sub, "B one")
		code, stdout, stderr := runDuty(t, sub, "get", "tracks", "--agent")
		if code != 0 || stderr != "" {
			t.Fatalf("get tracks --agent: code=%d stderr=%q", code, stderr)
		}
		oneLine(t, "stdout", stdout)
		if !strings.HasPrefix(stdout, ".\t") {
			t.Errorf("stdout = %q, want the backend board as \".\"", stdout)
		}
	})
}

func TestGetNext(t *testing.T) {
	t.Run("picks the first unblocked todo in board order", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "First")
		createTask(t, root, "Second")
		code, stdout, stderr := runDuty(t, root, "get", "next")
		if code != 0 || stderr != "" {
			t.Fatalf("get next: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "T-01") || strings.Contains(stdout, "T-02") {
			t.Errorf("stdout = %q, want T-01 only", stdout)
		}
	})

	t.Run("skips non-todo and todos blocked by an unfinished dependency", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Dep")
		mustRun(t, root, "status", "T-01", "in-progress")
		if code, _, stderr := runDuty(t, root, "create", "task", "Blocked", "--blocked-by", "T-01"); code != 0 {
			t.Fatalf("create: code=%d stderr=%q", code, stderr)
		}
		createTask(t, root, "Free")
		code, stdout, stderr := runDuty(t, root, "get", "next")
		if code != 0 || stderr != "" {
			t.Fatalf("get next: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "T-03") || strings.Contains(stdout, "T-01") || strings.Contains(stdout, "T-02") {
			t.Errorf("stdout = %q, want T-03 (T-01 in-progress, T-02 blocked)", stdout)
		}
	})

	t.Run("board position is priority, not the id order", func(t *testing.T) {
		root := initDuty(t)
		mustRun(t, root, "create", "track", "backend")
		sub := filepath.Join(root, "backend")
		createTask(t, sub, "Backend task")
		createTask(t, root, "Root task")
		code, stdout, stderr := runDuty(t, root, "get", "next")
		if code != 0 || stderr != "" {
			t.Fatalf("get next: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "T-02") || strings.Contains(stdout, "T-01") {
			t.Errorf("stdout = %q, want the root board's T-02 before the sub-track's T-01", stdout)
		}
	})

	t.Run("archived dependency counts as done", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old.md")
		code, _, stderr := runDuty(t, root, "create", "task", "Main", "--blocked-by", "T-90")
		if code != 0 {
			t.Fatalf("create: code=%d stderr=%q", code, stderr)
		}
		code, stdout, stderr := runDuty(t, root, "get", "next")
		if code != 0 || stderr != "" {
			t.Fatalf("get next: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "T-91") {
			t.Errorf("stdout = %q, want T-91 actionable (archived dep counts as done)", stdout)
		}
	})

	t.Run("reads file truth, not the board row status", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Only")
		boardPath := filepath.Join(root, "BOARD.md")
		desynced, err := board.SetRowStatus([]byte(readText(t, boardPath)), name, "done")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(boardPath, desynced, 0o644); err != nil {
			t.Fatal(err)
		}
		code, stdout, stderr := runDuty(t, root, "get", "next")
		if code != 0 || stderr != "" {
			t.Fatalf("get next: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "T-01") {
			t.Errorf("stdout = %q, want T-01 (file says todo though the board says done)", stdout)
		}
	})

	t.Run("empty output and exit 0 when nothing is actionable", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Only")
		mustRun(t, root, "status", "T-01", "done")
		code, stdout, stderr := runDuty(t, root, "get", "next")
		if code != 0 {
			t.Fatalf("get next: code=%d, want 0", code)
		}
		if stdout != "" || stderr != "" {
			t.Errorf("get next = stdout %q stderr %q, want both empty", stdout, stderr)
		}
	})

	t.Run("empty tree yields no output and exit 0", func(t *testing.T) {
		root := initDuty(t)
		code, stdout, stderr := runDuty(t, root, "get", "next")
		if code != 0 || stdout != "" || stderr != "" {
			t.Errorf("get next on an empty tree = (%d, %q, %q), want (0, \"\", \"\")", code, stdout, stderr)
		}
	})

	t.Run("--agent emits a single TSV record", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "First")
		code, stdout, stderr := runDuty(t, root, "get", "next", "--agent")
		if code != 0 || stderr != "" {
			t.Fatalf("get next --agent: code=%d stderr=%q", code, stderr)
		}
		oneLine(t, "stdout", stdout)
		fields := strings.Split(strings.TrimRight(stdout, "\n"), "\t")
		if len(fields) != 9 {
			t.Fatalf("record %q: got %d fields, want 9", stdout, len(fields))
		}
		if want := []string{"T-01", ".", "todo", "First", "0", "0", ""}; !equalFields(fields[:7], want) {
			t.Errorf("fields[:7] = %v, want %v", fields[:7], want)
		}
	})

	t.Run("rejects positional arguments", func(t *testing.T) {
		root := initDuty(t)
		code, _, stderr := runDuty(t, root, "get", "next", "extra")
		if code == 0 {
			t.Fatal("get next with a positional argument succeeded")
		}
		oneLine(t, "stderr", stderr)
	})
}
