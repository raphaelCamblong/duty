package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// treeWithTracks builds a fresh tree with tracks api/ and api/auth/ — three
// board levels (root, api, api/auth) — and returns the root. The nested track
// is created with `create track ... --in api`, exercising that path too.
func treeWithTracks(t *testing.T) string {
	t.Helper()
	root := initDuty(t)
	mustRunOut(t, root, "create", "track", "api")
	mustRunOut(t, root, "create", "track", "auth", "--in", "api")
	return root
}

// outside returns a directory outside the tree from which `./duty/` still
// resolves the root: the tree root's parent, where initDuty planted duty/.
func outside(t *testing.T, root string) string {
	t.Helper()
	return filepath.Dir(root)
}

func TestCreateTaskIn(t *testing.T) {
	t.Run("targets a nested board by path from outside the tree", func(t *testing.T) {
		root := treeWithTracks(t)
		code, stdout, stderr := runDuty(t, outside(t, root), "create", "task", "Deep", "--in", "api/auth")
		if code != 0 || stderr != "" {
			t.Fatalf("create task --in api/auth: code=%d stderr=%q", code, stderr)
		}
		_, got := splitCreateOutput(t, stdout)
		want := filepath.Join(root, "api", "auth")
		if filepath.Dir(got) != want {
			t.Errorf("task created in %q, want under %q", got, want)
		}
		if _, err := os.Stat(got); err != nil {
			t.Errorf("task file missing: %v", err)
		}
		board := readText(t, filepath.Join(root, "api", "auth", "BOARD.md"))
		if !strings.Contains(board, "("+filepath.Base(got)+")") {
			t.Errorf("nested board missing the row: %q", board)
		}
	})

	t.Run("--in . names the root board from a sub-track", func(t *testing.T) {
		root := treeWithTracks(t)
		code, stdout, stderr := runDuty(t, filepath.Join(root, "api", "auth"), "create", "task", "At root", "--in", ".")
		if code != 0 || stderr != "" {
			t.Fatalf("create task --in .: code=%d stderr=%q", code, stderr)
		}
		_, path := splitCreateOutput(t, stdout)
		if dir := filepath.Dir(path); dir != root {
			t.Errorf("task created in %q, want the root %q", dir, root)
		}
	})

	t.Run("unknown path errors and creates nothing", func(t *testing.T) {
		root := treeWithTracks(t)
		before := readText(t, filepath.Join(root, "BOARD.md"))
		code, stdout, stderr := runDuty(t, root, "create", "task", "Nope", "--in", "does/not/exist")
		if code != 1 {
			t.Fatalf("code=%d, want 1", code)
		}
		if stdout != "" {
			t.Errorf("stdout=%q, want empty", stdout)
		}
		if stderr != `unknown track "does/not/exist"`+"\n" {
			t.Errorf("stderr=%q, want unknown track line", stderr)
		}
		oneLine(t, "stderr", stderr)
		if readText(t, filepath.Join(root, "BOARD.md")) != before {
			t.Error("root board changed by a failed create")
		}
	})
}

func TestCreateTrackIn(t *testing.T) {
	t.Run("creates the sub-track under the --in board with its parent bullet", func(t *testing.T) {
		root := initDuty(t)
		mustRunOut(t, root, "create", "track", "api")
		mustRunOut(t, root, "create", "track", "billing", "--in", "api")

		if _, err := os.Stat(filepath.Join(root, "api", "billing", "BOARD.md")); err != nil {
			t.Errorf("sub-track board missing: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "api", "billing", "archive")); err != nil {
			t.Errorf("sub-track archive missing: %v", err)
		}
		parent := readText(t, filepath.Join(root, "api", "BOARD.md"))
		if !strings.Contains(parent, "[billing/](billing/BOARD.md)") {
			t.Errorf("parent board missing the boards bullet: %q", parent)
		}
		if strings.Contains(readText(t, filepath.Join(root, "BOARD.md")), "billing/") {
			t.Error("root board wrongly gained the sub-track bullet")
		}
	})

	t.Run("unknown --in path errors", func(t *testing.T) {
		root := initDuty(t)
		code, _, stderr := runDuty(t, root, "create", "track", "x", "--in", "ghost")
		if code != 1 {
			t.Fatalf("code=%d, want 1", code)
		}
		if stderr != `unknown track "ghost"`+"\n" {
			t.Errorf("stderr=%q", stderr)
		}
	})
}

func TestGetReadsIn(t *testing.T) {
	// seedSubtree plants one task directly in api/ and one in api/auth/, both
	// from outside the tree via --in, then returns the root.
	seed := func(t *testing.T) string {
		root := treeWithTracks(t)
		out := outside(t, root)
		mustRunOut(t, out, "create", "task", "Api task", "--in", "api")
		mustRunOut(t, out, "create", "task", "Auth task", "--in", "api/auth")
		return root
	}

	t.Run("get tasks --in scopes to the named subtree", func(t *testing.T) {
		root := seed(t)
		code, stdout, stderr := runDuty(t, outside(t, root), "get", "tasks", "--in", "api")
		if code != 0 || stderr != "" {
			t.Fatalf("get tasks --in api: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "Api task") || !strings.Contains(stdout, "Auth task") {
			t.Errorf("api subtree should list both tasks: %q", stdout)
		}

		_, deepOnly, _ := runDuty(t, outside(t, root), "get", "tasks", "--in", "api/auth")
		if strings.Contains(deepOnly, "Api task") || !strings.Contains(deepOnly, "Auth task") {
			t.Errorf("api/auth should list only the auth task: %q", deepOnly)
		}
	})

	t.Run("get tracks --in lists the subtree with the named board as .", func(t *testing.T) {
		root := seed(t)
		code, stdout, stderr := runDuty(t, outside(t, root), "get", "tracks", "--in", "api", "--agent")
		if code != 0 || stderr != "" {
			t.Fatalf("get tracks --in api: code=%d stderr=%q", code, stderr)
		}
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		if len(lines) != 2 {
			t.Fatalf("want the api board and its sub-track, got %q", stdout)
		}
		if !strings.HasPrefix(lines[0], ".\t") {
			t.Errorf("the --in board should head the list as \".\": %q", lines[0])
		}
		if !strings.HasPrefix(lines[1], "auth\t") {
			t.Errorf("second line should be the auth sub-track: %q", lines[1])
		}
	})

	t.Run("get next --in returns the first actionable task in the subtree", func(t *testing.T) {
		root := seed(t)
		code, stdout, stderr := runDuty(t, outside(t, root), "get", "next", "--in", "api/auth", "--agent")
		if code != 0 || stderr != "" {
			t.Fatalf("get next --in api/auth: code=%d stderr=%q", code, stderr)
		}
		if !strings.Contains(stdout, "Auth task") {
			t.Errorf("want the auth task, got %q", stdout)
		}
	})

	t.Run("unknown --in path errors on a read", func(t *testing.T) {
		root := treeWithTracks(t)
		code, stdout, stderr := runDuty(t, root, "get", "tasks", "--in", "missing")
		if code != 1 || stdout != "" {
			t.Fatalf("code=%d stdout=%q, want error and no output", code, stdout)
		}
		if stderr != `unknown track "missing"`+"\n" {
			t.Errorf("stderr=%q", stderr)
		}
	})
}

func TestArchiveIn(t *testing.T) {
	t.Run("archives done tasks in the --in board from outside the tree", func(t *testing.T) {
		root := treeWithTracks(t)
		out := outside(t, root)
		mustRunOut(t, out, "create", "task", "Ship it", "--in", "api")
		mustRunOut(t, out, "status", "T-01", "done")

		mustRunOut(t, out, "archive", "--in", "api")

		if names, _ := os.ReadDir(filepath.Join(root, "api", "archive")); len(names) != 1 {
			t.Errorf("want one archived file under api/archive, got %d", len(names))
		}
		board := readText(t, filepath.Join(root, "api", "BOARD.md"))
		if strings.Contains(board, "Ship it") {
			t.Errorf("api board still lists the archived task: %q", board)
		}
		if !strings.Contains(board, "Completed tasks (1)") {
			t.Errorf("api board footer count not updated: %q", board)
		}
	})

	t.Run("unknown --in path errors", func(t *testing.T) {
		root := treeWithTracks(t)
		code, _, stderr := runDuty(t, root, "archive", "--in", "nope")
		if code != 1 {
			t.Fatalf("code=%d, want 1", code)
		}
		if stderr != `unknown track "nope"`+"\n" {
			t.Errorf("stderr=%q", stderr)
		}
	})
}

// mustRunOut runs a duty command from dir that must succeed; unlike mustRun it
// tolerates stdout (create prints the new path) and only fails on a non-zero
// code or any stderr.
func mustRunOut(t *testing.T, dir string, args ...string) {
	t.Helper()
	code, _, stderr := runDuty(t, dir, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("duty %v: code=%d stderr=%q", args, code, stderr)
	}
}
