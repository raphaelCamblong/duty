package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
)

// snapshotTree captures the content of every regular file under root, keyed
// by its path relative to root, for before/after comparison.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == names.LockFile {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		snap[rel] = readText(t, path)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return snap
}

func TestArchive(t *testing.T) {
	t.Run("archives a done task into its own board's archive, dropping and pruning its row", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "First task")
		mustRun(t, root, "status", "T-01", "done")
		mustRun(t, root, "archive")

		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("done task still at board top level (err %v)", err)
		}
		archived := filepath.Join(root, "archive", name)
		if _, err := os.Stat(archived); err != nil {
			t.Errorf("done task not archived: %v", err)
		}
		got := readText(t, filepath.Join(root, "BOARD.md"))
		if strings.Contains(got, "("+name+")") {
			t.Errorf("board still has the archived row: %q", got)
		}
		if !strings.Contains(got, "Completed tasks (1) archived") {
			t.Errorf("footer count not rewritten: %q", got)
		}
	})

	t.Run("archives sub-board tasks into that sub-board's own archive, not the root's", func(t *testing.T) {
		root := initDuty(t)
		mustRunOut(t, root, "create", "track", "backend")
		sub := filepath.Join(root, "backend")
		rootName := createTask(t, root, "Root task")
		subName := createTask(t, sub, "Backend task")
		mustRun(t, root, "status", "T-01", "done")
		mustRun(t, root, "status", "T-02", "done")
		mustRun(t, root, "archive")

		if _, err := os.Stat(filepath.Join(root, "archive", rootName)); err != nil {
			t.Errorf("root task not in root archive: %v", err)
		}
		if _, err := os.Stat(filepath.Join(sub, "archive", subName)); err != nil {
			t.Errorf("backend task not in backend archive: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "archive", subName)); !os.IsNotExist(err) {
			t.Errorf("backend task leaked into root archive (err %v)", err)
		}
		if _, err := os.Stat(filepath.Join(sub, "archive", rootName)); !os.IsNotExist(err) {
			t.Errorf("root task leaked into backend archive (err %v)", err)
		}
		if got := readText(t, filepath.Join(sub, "BOARD.md")); !strings.Contains(got, "Completed tasks (1) archived") {
			t.Errorf("backend footer count not rewritten: %q", got)
		}
	})

	t.Run("leaves non-done tasks and their rows untouched", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "In progress task")
		mustRun(t, root, "status", "T-01", "in-progress")
		before := readText(t, filepath.Join(root, "BOARD.md"))
		mustRun(t, root, "archive")
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("non-done task archived away: %v", err)
		}
		if got := readText(t, filepath.Join(root, "BOARD.md")); got != before {
			t.Errorf("board changed archiving with nothing done:\n got %q\nwant %q", got, before)
		}
	})

	t.Run("nothing to archive is a clean quiet no-op", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Still open")
		mustRun(t, root, "archive")
	})

	t.Run("a second run is a no-op leaving the tree byte-identical", func(t *testing.T) {
		root := initDuty(t)
		mustRunOut(t, root, "create", "track", "backend")
		createTask(t, root, "Root task")
		createTask(t, filepath.Join(root, "backend"), "Backend task")
		mustRun(t, root, "status", "T-01", "done")
		mustRun(t, root, "status", "T-02", "done")
		mustRun(t, root, "archive")

		before := snapshotTree(t, root)
		mustRun(t, root, "archive")
		after := snapshotTree(t, root)

		if len(before) != len(after) {
			t.Fatalf("file count changed: before %d, after %d", len(before), len(after))
		}
		for path, content := range before {
			if after[path] != content {
				t.Errorf("%s changed by idempotent second archive", path)
			}
		}
	})

	t.Run("only archives the current board and below, not a sibling", func(t *testing.T) {
		root := initDuty(t)
		mustRunOut(t, root, "create", "track", "backend")
		sub := filepath.Join(root, "backend")
		rootName := createTask(t, root, "Root task")
		mustRun(t, root, "status", "T-01", "done")
		mustRun(t, sub, "archive")
		if _, err := os.Stat(filepath.Join(root, rootName)); err != nil {
			t.Errorf("root task archived by a sub-board-scoped archive: %v", err)
		}
	})

	t.Run("rejects extra arguments", func(t *testing.T) {
		root := initDuty(t)
		code, _, stderr := runDuty(t, root, "archive", "extra")
		if code == 0 {
			t.Fatal("archive with an argument succeeded")
		}
		oneLine(t, "stderr", stderr)
	})
}

func TestDelete(t *testing.T) {
	t.Run("refuses a done task without --force, changing nothing", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Finished task")
		mustRun(t, root, "status", "T-01", "done")
		boardBefore := readText(t, filepath.Join(root, "BOARD.md"))
		code, stdout, stderr := runDuty(t, root, "delete", "task", "T-01")
		if code == 0 {
			t.Fatal("delete of a done task without --force succeeded")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "done") {
			t.Errorf("stderr = %q, want it to mention done", stderr)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("file removed despite refusal: %v", err)
		}
		if got := readText(t, filepath.Join(root, "BOARD.md")); got != boardBefore {
			t.Error("board changed by refused delete")
		}
	})

	t.Run("--force deletes a done task", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Finished task")
		mustRun(t, root, "status", "T-01", "done")
		mustRun(t, root, "delete", "task", "T-01", "--force")
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("file still present (err %v)", err)
		}
		if got := readText(t, filepath.Join(root, "BOARD.md")); strings.Contains(got, "("+name+")") {
			t.Errorf("board still has the row: %q", got)
		}
	})

	t.Run("deletes an open task and prunes its emptied section", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Open task")
		mustRun(t, root, "move", "T-01", "--section", "Later")
		mustRun(t, root, "delete", "task", "T-01")
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("file still present (err %v)", err)
		}
		got := readText(t, filepath.Join(root, "BOARD.md"))
		if strings.Contains(got, "## Later") {
			t.Errorf("emptied section not pruned: %q", got)
		}
	})

	t.Run("rejects archived ids", func(t *testing.T) {
		root := initDuty(t)
		writeArchived(t, root, "T-90-old-work.md")
		code, _, stderr := runDuty(t, root, "delete", "task", "T-90", "--force")
		if code == 0 {
			t.Fatal("delete on an archived id succeeded")
		}
		oneLine(t, "stderr", stderr)
		if !strings.Contains(stderr, "archived") {
			t.Errorf("stderr = %q, want it to say archived", stderr)
		}
	})

	t.Run("unknown id errors", func(t *testing.T) {
		root := initDuty(t)
		code, _, stderr := runDuty(t, root, "delete", "task", "T-99")
		if code == 0 {
			t.Fatal("delete on an unknown id succeeded")
		}
		oneLine(t, "stderr", stderr)
	})

	t.Run("argument validation", func(t *testing.T) {
		for _, args := range [][]string{{"delete", "task"}, {"delete", "task", "T-01", "extra"}} {
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

func TestGetTasks(t *testing.T) {
	t.Run("lists local open tasks from the files, no prefix", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "First task")
		mustRun(t, root, "status", "T-01", "in-progress")
		code, stdout, stderr := runDuty(t, root, "get", "tasks")
		if code != 0 || stderr != "" {
			t.Fatalf("get tasks: code=%d stderr=%q", code, stderr)
		}
		if strings.Contains(stdout, "T-01/") || strings.Contains(stdout, "/ T-01") {
			t.Errorf("local task got a sub-board prefix: %q", stdout)
		}
		idAt := strings.Index(stdout, "T-01")
		statusAt := strings.Index(stdout, "in-progress")
		titleAt := strings.Index(stdout, "First task")
		if idAt < 0 || statusAt < idAt || titleAt < statusAt {
			t.Errorf("want id then status then title, got %q", stdout)
		}
	})

	t.Run("prefixes rows from a sub-board with its path", func(t *testing.T) {
		root := initDuty(t)
		mustRunOut(t, root, "create", "track", "backend")
		createTask(t, filepath.Join(root, "backend"), "Backend task")
		code, stdout, stderr := runDuty(t, root, "get", "tasks")
		if code != 0 || stderr != "" {
			t.Fatalf("get tasks: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "backend/ T-01") {
			t.Errorf("stdout = %q, want a %q prefix", stdout, "backend/ T-01")
		}
	})

	t.Run("filters by --status", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Todo task")
		createTask(t, root, "Doing task")
		mustRun(t, root, "status", "T-02", "in-progress")
		code, stdout, stderr := runDuty(t, root, "get", "tasks", "--status", "in-progress")
		if code != 0 || stderr != "" {
			t.Fatalf("get tasks: code=%d stderr=%q", code, stderr)
		}
		if strings.Contains(stdout, "T-01") || !strings.Contains(stdout, "T-02") {
			t.Errorf("stdout = %q, want only T-02", stdout)
		}
	})

	t.Run("rejects an unknown --status", func(t *testing.T) {
		root := initDuty(t)
		code, stdout, stderr := runDuty(t, root, "get", "tasks", "--status", "cancelled")
		if code == 0 {
			t.Fatal("get tasks with an unknown status succeeded")
		}
		oneLine(t, "stderr", stderr)
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})

	t.Run("flags a status mismatch against a hand-edited row, without touching either file", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Drifted task")
		boardPath := filepath.Join(root, "BOARD.md")
		index := readText(t, boardPath)
		desynced, err := board.SetRowStatus([]byte(index), name, "done")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(boardPath, desynced, 0o644); err != nil {
			t.Fatal(err)
		}
		taskBefore := readText(t, filepath.Join(root, name))
		boardBefore := readText(t, boardPath)

		code, stdout, stderr := runDuty(t, root, "get", "tasks")
		if code != 0 || stderr != "" {
			t.Fatalf("get tasks: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "⚠ board says done") {
			t.Errorf("stdout = %q, want a drift flag naming the board's status", stdout)
		}
		if readText(t, filepath.Join(root, name)) != taskBefore {
			t.Error("task file changed by get tasks")
		}
		if readText(t, boardPath) != boardBefore {
			t.Error("board changed by get tasks")
		}
	})

	t.Run("flags a missing row without touching either file", func(t *testing.T) {
		root := initDuty(t)
		name := createTask(t, root, "Row-less task")
		boardPath := filepath.Join(root, "BOARD.md")
		index := readText(t, boardPath)
		dropped, err := board.DropRow([]byte(index), name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(boardPath, dropped, 0o644); err != nil {
			t.Fatal(err)
		}
		taskBefore := readText(t, filepath.Join(root, name))
		boardBefore := readText(t, boardPath)

		code, stdout, stderr := runDuty(t, root, "get", "tasks")
		if code != 0 || stderr != "" {
			t.Fatalf("get tasks: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "T-01") || !strings.Contains(stdout, "⚠") {
			t.Errorf("stdout = %q, want T-01 flagged", stdout)
		}
		if readText(t, filepath.Join(root, name)) != taskBefore {
			t.Error("task file changed by get tasks")
		}
		if readText(t, boardPath) != boardBefore {
			t.Error("board changed by get tasks")
		}
	})

	t.Run("rejects extra arguments", func(t *testing.T) {
		root := initDuty(t)
		code, _, stderr := runDuty(t, root, "get", "tasks", "extra")
		if code == 0 {
			t.Fatal("get tasks with a positional argument succeeded")
		}
		oneLine(t, "stderr", stderr)
	})

	t.Run("--agent emits exact TSV field order, in sync and drifted", func(t *testing.T) {
		root := initDuty(t)
		mustRunOut(t, root, "create", "track", "backend")
		sub := filepath.Join(root, "backend")
		rootName := createTask(t, root, "Root task")
		createTask(t, sub, "Backend task")

		boardPath := filepath.Join(root, "BOARD.md")
		index := readText(t, boardPath)
		desynced, err := board.SetRowStatus([]byte(index), rootName, "done")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(boardPath, desynced, 0o644); err != nil {
			t.Fatal(err)
		}

		code, stdout, stderr := runDuty(t, root, "get", "tasks", "--agent")
		if code != 0 || stderr != "" {
			t.Fatalf("get tasks --agent: code=%d stderr=%q", code, stderr)
		}
		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		records := make(map[string][]string, len(lines))
		for _, line := range lines {
			fields := strings.Split(line, "\t")
			if len(fields) != 8 {
				t.Fatalf("record %q: got %d fields, want 8", line, len(fields))
			}
			records[fields[0]] = fields
		}

		root01, ok := records["T-01"]
		if !ok {
			t.Fatalf("no record for T-01 in %q", stdout)
		}
		if want := []string{"T-01", ".", "todo", "Root task", "board=done"}; !equalFields(root01[:5], want) {
			t.Errorf("T-01 record = %v, want %v", root01[:5], want)
		}

		sub02, ok := records["T-02"]
		if !ok {
			t.Fatalf("no record for T-02 in %q", stdout)
		}
		if want := []string{"T-02", "backend", "todo", "Backend task", ""}; !equalFields(sub02[:5], want) {
			t.Errorf("T-02 record = %v, want %v", sub02[:5], want)
		}
	})

	t.Run("hidden list alias matches get tasks output", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "First task")
		gotCode, gotOut, gotErr := runDuty(t, root, "get", "tasks")
		aliasCode, aliasOut, aliasErr := runDuty(t, root, "list")
		if aliasCode != gotCode || aliasOut != gotOut || aliasErr != gotErr {
			t.Errorf("list = (%d, %q, %q), want get tasks' (%d, %q, %q)",
				aliasCode, aliasOut, aliasErr, gotCode, gotOut, gotErr)
		}
		if code, _, stderr := runDuty(t, root, "list", "--agent"); code != 0 || stderr != "" {
			t.Errorf("list --agent: code=%d stderr=%q", code, stderr)
		}
	})

	t.Run("lists an unparsable file as a bad-file row instead of failing", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Good task")
		if err := os.WriteFile(filepath.Join(root, "T-08-bad.md"), []byte("not valid frontmatter\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		boardPath := filepath.Join(root, "BOARD.md")
		updated, err := board.AddRow([]byte(readText(t, boardPath)), board.DefaultSection,
			board.Row{ID: "T-08", File: "T-08-bad.md", Title: "Bad one", Status: "todo"})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(boardPath, updated, 0o644); err != nil {
			t.Fatal(err)
		}

		code, human, stderr := runDuty(t, root, "get", "tasks")
		if code != 0 || stderr != "" {
			t.Fatalf("get tasks with a bad file: code=%d stderr=%q, want it listed not an error", code, stderr)
		}
		if !strings.Contains(human, "T-08") || !strings.Contains(human, "⚠ unparsable") {
			t.Errorf("human = %q, want T-08 flagged unparsable", human)
		}
		rec := tasksRecord(t, root, "T-08")
		if rec[4] != "bad-file" {
			t.Errorf("drift field = %q, want bad-file", rec[4])
		}
		if rec[2] != "todo" {
			t.Errorf("status field = %q, want todo from board truth", rec[2])
		}
	})

	t.Run("lists a board row with no file, its status board truth", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Real task")
		boardPath := filepath.Join(root, "BOARD.md")
		updated, err := board.AddRow([]byte(readText(t, boardPath)), board.DefaultSection,
			board.Row{ID: "T-09", File: "T-09-ghost.md", Title: "Ghost", Status: "blocked"})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(boardPath, updated, 0o644); err != nil {
			t.Fatal(err)
		}

		rec := tasksRecord(t, root, "T-09")
		if rec[4] != "no-file" {
			t.Errorf("drift field = %q, want no-file", rec[4])
		}
		if rec[2] != "blocked" {
			t.Errorf("status field = %q, want blocked from board truth", rec[2])
		}
		_, human, _ := runDuty(t, root, "get", "tasks")
		if !strings.Contains(human, "⚠ no file") {
			t.Errorf("human = %q, want a no-file flag", human)
		}
	})

	t.Run("--agent trails claimed-by then waits", func(t *testing.T) {
		root := initDuty(t)
		createTask(t, root, "Dep")
		mustRun(t, root, "status", "T-01", "in-progress", "--as", "sonnet-9")
		mustDuty(t, root, "create", "task", "Waiter", "--blocked-by", "T-01")

		dep := tasksRecord(t, root, "T-01")
		if len(dep) != 8 {
			t.Fatalf("record %v: got %d fields, want 8", dep, len(dep))
		}
		if dep[6] != "sonnet-9" || dep[7] != "" {
			t.Errorf("dep record claimed-by/waits = %q/%q, want sonnet-9/empty", dep[6], dep[7])
		}
		waiter := tasksRecord(t, root, "T-02")
		if waiter[6] != "" || waiter[7] != "T-01" {
			t.Errorf("waiter record claimed-by/waits = %q/%q, want empty/T-01", waiter[6], waiter[7])
		}
	})
}

// tasksRecord runs `get tasks --agent` and returns the TSV fields of the row
// naming id, failing when no row does.
func tasksRecord(t *testing.T, root, id string) []string {
	t.Helper()
	code, stdout, stderr := runDuty(t, root, "get", "tasks", "--agent")
	if code != 0 || stderr != "" {
		t.Fatalf("get tasks --agent: code=%d stderr=%q", code, stderr)
	}
	for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
		fields := strings.Split(line, "\t")
		if fields[0] == id {
			return fields
		}
	}
	t.Fatalf("no record for %s in %q", id, stdout)
	return nil
}

// equalFields reports whether got and want hold the same strings in order.
func equalFields(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
