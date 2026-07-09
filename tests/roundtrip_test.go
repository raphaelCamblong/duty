package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/task"
)

// The fixtures below are hand-written boards salted with prose, banners, and
// odd spacing the CLI does not own. Every byte is load-bearing: the tests
// assert full-tree byte equality against expectations derived from these
// constants, so any writer that rewrites more than its target line fails.
const (
	tableHead = "| Task | Title | Status |"
	tableSep  = "|------|-------|--------|"

	saltT01Row = "| [T-01](T-01-existing-task.md) | Existing task | todo |"
	saltT02Row = "|  [T-02](T-02-odd-spacing.md)  |   Odd   spacing   |  in-progress  |"
	saltT03Row = "| [T-03](T-03-parked-task.md) | Parked task | blocked |"
	saltBullet = "- keep: a hand-written bullet the tooling never reads"

	// normalT02Row is T-02's row as the CLI re-renders it from the task file
	// when moving across boards: the moved row is the target line, so its
	// hand padding does not survive — everything around it must.
	normalT02Row = "| [T-02](T-02-odd-spacing.md) | Odd   spacing | in-progress |"
)

const saltedBoard = "# Gnarly board\n" +
	"\n" +
	"> BANNER: hand-written — the CLI must never touch this line.\n" +
	"\n" +
	"Prose with   odd    spacing kept exactly as written.\n" +
	"<!-- an HTML comment the tooling never parses -->\n" +
	"\n" +
	"## Boards\n" +
	"\n" +
	"- [backend/](backend/BOARD.md) — Backend\n" +
	saltBullet + "\n" +
	"\n" +
	"## Open tasks\n" +
	"\n" +
	"Intro prose inside the default section.\n" +
	"\n" +
	tableHead + "\n" +
	"|:-----|:------|:-------|\n" +
	saltT01Row + "\n" +
	saltT02Row + "\n" +
	"\n" +
	"Trailing note after the table.\n" +
	"\n" +
	"## Parked\n" +
	"\n" +
	tableHead + "\n" +
	tableSep + "\n" +
	saltT03Row + "\n" +
	"\n" +
	"Completed tasks (2) archived: [archive/](archive/).\n" +
	"\n" +
	"Postscript below the footer — also sacred.\n"

const saltedBackend = "# Backend\n" +
	"\n" +
	"Hand note: the backend board keeps its own prose.\n" +
	"\n" +
	"## Open tasks\n" +
	"\n" +
	tableHead + "\n" +
	tableSep + "\n" +
	"\n" +
	"Completed tasks (0) archived: [archive/](archive/).\n"

const saltT01File = "---\n" +
	"id: T-01\n" +
	"title: Existing task\n" +
	"status: todo\n" +
	"blocked-by: []\n" +
	"---\n" +
	"\n" +
	"# T-01 — Existing task\n" +
	"\n" +
	"## Gates\n" +
	"- [ ] Untouched by any command.\n" +
	"\n" +
	"## Report\n"

const saltT02File = "---\n" +
	"id: T-02\n" +
	"title: Odd   spacing\n" +
	"status: in-progress\n" +
	"blocked-by: []\n" +
	"---\n" +
	"\n" +
	"# T-02 — Odd   spacing\n" +
	"\n" +
	"## Gates\n" +
	"- [x] Ticked by hand.\n" +
	"- [ ] Still open.\n" +
	"\n" +
	"## Report\n" +
	"\n" +
	"Hand-written first report.\n"

const saltT03File = "---\n" +
	"id: T-03\n" +
	"title: Parked task\n" +
	"status: blocked\n" +
	"blocked-by: [T-01]\n" +
	"---\n" +
	"\n" +
	"# T-03 — Parked task\n" +
	"\n" +
	"## Report\n" +
	"\n" +
	"Blocked on a decision the task does not pre-make.\n"

// writeSalted builds the salted fixture tree in a temp dir and returns its
// root: the salted root board with three tasks, two archived tasks (so the
// next tree-wide number is T-06), and a backend sub-board.
func writeSalted(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "duty")
	for _, dir := range []string{
		filepath.Join(root, "archive"),
		filepath.Join(root, "backend", "archive"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"BOARD.md":              saltedBoard,
		"T-01-existing-task.md": saltT01File,
		"T-02-odd-spacing.md":   saltT02File,
		"T-03-parked-task.md":   saltT03File,
		"archive/T-04-shipped.md": "---\nid: T-04\ntitle: Shipped\nstatus: done\nblocked-by: []\n---\n" +
			"\n# T-04 — Shipped\n",
		"archive/T-05-also-shipped.md": "---\nid: T-05\ntitle: Also shipped\nstatus: done\nblocked-by: []\n---\n" +
			"\n# T-05 — Also shipped\n",
		"backend/BOARD.md": saltedBackend,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// hashTree hashes every directory name and file content under root, in
// walk order — the spec §6 before/after fingerprint.
func hashTree(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			fmt.Fprintf(h, "dir %s\n", filepath.ToSlash(rel))
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "file %s %d\n", filepath.ToSlash(rel), len(data))
		h.Write(data)
		return nil
	})
	if err != nil {
		t.Fatalf("hash %s: %v", root, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// diffTrees fails on every file that differs between want and got, naming
// changed, missing, and unexpected files.
func diffTrees(t *testing.T, want, got map[string]string) {
	t.Helper()
	for path, w := range want {
		g, ok := got[path]
		if !ok {
			t.Errorf("%s: file missing", path)
			continue
		}
		if g != w {
			t.Errorf("%s:\n got %q\nwant %q", path, g, w)
		}
	}
	for path := range got {
		if _, ok := want[path]; !ok {
			t.Errorf("%s: unexpected file", path)
		}
	}
}

// replaceOnce replaces the single occurrence of old in s, guarding the
// fixture: old appearing zero or several times means the expectation would
// be built on drifted constants.
func replaceOnce(t *testing.T, s, old, new string) string {
	t.Helper()
	if n := strings.Count(s, old); n != 1 {
		t.Fatalf("fixture: %q appears %d times, want exactly 1", old, n)
	}
	return strings.Replace(s, old, new, 1)
}

// TestRoundTrip is the master acceptance test (spec §6): the full lifecycle
// of a scratch task on a salted tree leaves every byte of the tree exactly
// as it was — proof that every writer preserves everything it doesn't own.
func TestRoundTrip(t *testing.T) {
	root := writeSalted(t)
	beforeHash := hashTree(t, root)
	before := snapshotTree(t, root)

	code, stdout, stderr := runDuty(t, root, "create", "Scratch pad")
	if code != 0 || stderr != "" {
		t.Fatalf("create: code=%d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "T-06-scratch-pad.md") {
		t.Fatalf("create printed %q, want the T-06 path", stdout)
	}
	if hashTree(t, root) == beforeHash {
		t.Fatal("hash unchanged after create: hashTree cannot see changes")
	}

	mustRun(t, root, "status", "T-06", "in-progress")
	if code, stdout, stderr := runDutyStdin(t, root, "Scratch report.\n", "report", "T-06"); code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("report: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	mustRun(t, root, "link", "T-06", "Waiting")
	mustRun(t, root, "move", "T-06", "backend")
	mustRun(t, filepath.Join(root, "backend"), "move", "T-06", ".")
	mustRun(t, root, "delete", "T-06")
	mustRun(t, root, "archive")

	if got := hashTree(t, root); got != beforeHash {
		diffTrees(t, before, snapshotTree(t, root))
		t.Errorf("tree hash after round-trip = %s, want %s", got, beforeHash)
	}
}

// TestSaltedBoardSurvivesEveryMutation runs each mutating command against
// the salted fixture and asserts every file in the tree byte-for-byte:
// outside the lines a command owns, nothing may change.
func TestSaltedBoardSurvivesEveryMutation(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, root string)
		want func(t *testing.T, want map[string]string)
	}{
		{
			name: "status rewrites one status cell and one frontmatter line",
			run: func(t *testing.T, root string) {
				mustRun(t, root, "status", "T-02", "done")
			},
			want: func(t *testing.T, w map[string]string) {
				w["BOARD.md"] = replaceOnce(t, saltedBoard, saltT02Row,
					"|  [T-02](T-02-odd-spacing.md)  |   Odd   spacing   | done |")
				w["T-02-odd-spacing.md"] = replaceOnce(t, saltT02File, "status: in-progress", "status: done")
			},
		},
		{
			name: "create appends one row after the last table line",
			run: func(t *testing.T, root string) {
				code, stdout, stderr := runDuty(t, root, "create", "Fresh work")
				if code != 0 || stderr != "" {
					t.Fatalf("create: code=%d stderr=%q", code, stderr)
				}
				if !strings.Contains(stdout, "T-06-fresh-work.md") {
					t.Fatalf("create printed %q, want the T-06 path", stdout)
				}
			},
			want: func(t *testing.T, w map[string]string) {
				w["BOARD.md"] = replaceOnce(t, saltedBoard, saltT02Row+"\n",
					saltT02Row+"\n| [T-06](T-06-fresh-work.md) | Fresh work | todo |\n")
				w["T-06-fresh-work.md"] = string(task.Render("T-06", "Fresh work", nil))
			},
		},
		{
			name: "report appends to the task file and touches nothing else",
			run: func(t *testing.T, root string) {
				code, stdout, stderr := runDutyStdin(t, root, "Second report.\n", "report", "T-02")
				if code != 0 || stdout != "" || stderr != "" {
					t.Fatalf("report: code=%d stdout=%q stderr=%q", code, stdout, stderr)
				}
			},
			want: func(t *testing.T, w map[string]string) {
				w["T-02-odd-spacing.md"] = saltT02File + "\nSecond report.\n"
			},
		},
		{
			name: "link into an existing section moves only the row line",
			run: func(t *testing.T, root string) {
				mustRun(t, root, "link", "T-01", "Parked")
			},
			want: func(t *testing.T, w map[string]string) {
				b := replaceOnce(t, saltedBoard, saltT01Row+"\n", "")
				w["BOARD.md"] = replaceOnce(t, b, saltT03Row+"\n", saltT03Row+"\n"+saltT01Row+"\n")
			},
		},
		{
			name: "link creating a section prunes the emptied one and nothing more",
			run: func(t *testing.T, root string) {
				mustRun(t, root, "link", "T-03", "Waiting")
			},
			want: func(t *testing.T, w map[string]string) {
				w["BOARD.md"] = replaceOnce(t, saltedBoard, "## Parked\n", "## Waiting\n")
			},
		},
		{
			name: "move touches one row per board and relocates the file verbatim",
			run: func(t *testing.T, root string) {
				mustRun(t, root, "move", "T-02", "backend")
			},
			want: func(t *testing.T, w map[string]string) {
				w["BOARD.md"] = replaceOnce(t, saltedBoard, saltT02Row+"\n", "")
				w[filepath.FromSlash("backend/BOARD.md")] = replaceOnce(t, saltedBackend,
					tableSep+"\n", tableSep+"\n"+normalT02Row+"\n")
				delete(w, "T-02-odd-spacing.md")
				w[filepath.FromSlash("backend/T-02-odd-spacing.md")] = saltT02File
			},
		},
		{
			name: "move there and back restores everything but the moved row's hand padding",
			run: func(t *testing.T, root string) {
				mustRun(t, root, "move", "T-02", "backend")
				mustRun(t, filepath.Join(root, "backend"), "move", "T-02", ".")
			},
			want: func(t *testing.T, w map[string]string) {
				w["BOARD.md"] = replaceOnce(t, saltedBoard, saltT02Row, normalT02Row)
			},
		},
		{
			name: "delete removes only the file and its row",
			run: func(t *testing.T, root string) {
				mustRun(t, root, "delete", "T-01")
			},
			want: func(t *testing.T, w map[string]string) {
				w["BOARD.md"] = replaceOnce(t, saltedBoard, saltT01Row+"\n", "")
				delete(w, "T-01-existing-task.md")
			},
		},
		{
			name: "archive moves the done task, prunes its section, recounts the footer",
			run: func(t *testing.T, root string) {
				mustRun(t, root, "status", "T-03", "done")
				mustRun(t, root, "archive")
			},
			want: func(t *testing.T, w map[string]string) {
				b := replaceOnce(t, saltedBoard,
					"## Parked\n\n"+tableHead+"\n"+tableSep+"\n"+saltT03Row+"\n\n", "")
				w["BOARD.md"] = replaceOnce(t, b, "Completed tasks (2)", "Completed tasks (3)")
				delete(w, "T-03-parked-task.md")
				w[filepath.FromSlash("archive/T-03-parked-task.md")] =
					replaceOnce(t, saltT03File, "status: blocked", "status: done")
			},
		},
		{
			name: "board appends one bullet after the last hand-written one",
			run: func(t *testing.T, root string) {
				mustRun(t, root, "board", "api")
			},
			want: func(t *testing.T, w map[string]string) {
				w["BOARD.md"] = replaceOnce(t, saltedBoard, saltBullet+"\n",
					saltBullet+"\n- [api/](api/BOARD.md) — api\n")
				w[filepath.FromSlash("api/BOARD.md")] = string(board.Render("api"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeSalted(t)
			want := snapshotTree(t, root)
			tt.run(t, root)
			tt.want(t, want)
			diffTrees(t, want, snapshotTree(t, root))
		})
	}
}

const pruneRow = "| [T-01](T-01-only-task.md) | Only task | todo |"

const pruneBoard = "# Prune board\n" +
	"\n" +
	"## Open tasks\n" +
	"\n" +
	tableHead + "\n" +
	tableSep + "\n" +
	"\n" +
	"## Later\n" +
	"\n" +
	tableHead + "\n" +
	tableSep + "\n" +
	pruneRow + "\n" +
	"\n" +
	"Completed tasks (0) archived: [archive/](archive/).\n"

const pruneTask = "---\nid: T-01\ntitle: Only task\nstatus: todo\nblocked-by: []\n---\n" +
	"\n# T-01 — Only task\n\n## Report\n"

// writePruneTree builds a board whose default section is already empty and
// whose only task sits in "## Later".
func writePruneTree(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "duty")
	if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"BOARD.md":          pruneBoard,
		"T-01-only-task.md": pruneTask,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestPruneNeverRemovesDefaultSection: emptying every table may prune
// "## Later" but the default section survives, however empty (spec §6).
func TestPruneNeverRemovesDefaultSection(t *testing.T) {
	laterBlock := "## Later\n\n" + tableHead + "\n" + tableSep + "\n" + pruneRow + "\n\n"

	t.Run("delete prunes the emptied section, keeps the empty default", func(t *testing.T) {
		root := writePruneTree(t)
		want := snapshotTree(t, root)
		mustRun(t, root, "delete", "T-01")
		want["BOARD.md"] = replaceOnce(t, pruneBoard, laterBlock, "")
		delete(want, "T-01-only-task.md")
		diffTrees(t, want, snapshotTree(t, root))
		if got := readText(t, filepath.Join(root, "BOARD.md")); !strings.Contains(got, "## Open tasks") {
			t.Errorf("default section pruned: %q", got)
		}
	})

	t.Run("archive prunes the emptied section, keeps the empty default", func(t *testing.T) {
		root := writePruneTree(t)
		want := snapshotTree(t, root)
		mustRun(t, root, "status", "T-01", "done")
		mustRun(t, root, "archive")
		b := replaceOnce(t, pruneBoard, laterBlock, "")
		want["BOARD.md"] = replaceOnce(t, b, "Completed tasks (0)", "Completed tasks (1)")
		delete(want, "T-01-only-task.md")
		want[filepath.FromSlash("archive/T-01-only-task.md")] =
			replaceOnce(t, pruneTask, "status: todo", "status: done")
		diffTrees(t, want, snapshotTree(t, root))
		if got := readText(t, filepath.Join(root, "BOARD.md")); !strings.Contains(got, "## Open tasks") {
			t.Errorf("default section pruned: %q", got)
		}
	})
}

// TestListNeverWrites: list reads files as truth and reports drift without
// touching a single byte, in every output mode (spec §6).
func TestListNeverWrites(t *testing.T) {
	root := writeSalted(t)
	drifted := replaceOnce(t, saltedBoard, saltT01Row,
		"| [T-01](T-01-existing-task.md) | Existing task | done |")
	drifted = replaceOnce(t, drifted, saltT03Row+"\n", "")
	if err := os.WriteFile(filepath.Join(root, "BOARD.md"), []byte(drifted), 0o644); err != nil {
		t.Fatal(err)
	}
	before := hashTree(t, root)
	snap := snapshotTree(t, root)

	code, stdout, stderr := runDuty(t, root, "list")
	if code != 0 || stderr != "" {
		t.Fatalf("list: code=%d stderr=%q", code, stderr)
	}
	for _, want := range []string{"⚠ board says done", "⚠ board says missing"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list output %q missing drift flag %q", stdout, want)
		}
	}
	for _, args := range [][]string{
		{"list", "--agent"},
		{"list", "--status", "blocked"},
		{"list", "--status", "done"},
	} {
		if code, _, stderr := runDuty(t, root, args...); code != 0 || stderr != "" {
			t.Fatalf("duty %v: code=%d stderr=%q", args, code, stderr)
		}
	}

	if got := hashTree(t, root); got != before {
		diffTrees(t, snap, snapshotTree(t, root))
		t.Errorf("tree hash after list = %s, want %s", got, before)
	}
}
