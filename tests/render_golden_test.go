package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/task"
)

// goldenFile reads tests/testdata/name, failing the test on error.
func goldenFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(data)
}

func TestRenderGoldens(t *testing.T) {
	tests := []struct {
		name   string
		got    []byte
		golden string
	}{
		{
			name:   "task skeleton without blocked-by",
			got:    task.Render("T-01", "First task", nil),
			golden: "task_skeleton.md",
		},
		{
			name:   "task skeleton with blocked-by",
			got:    task.Render("T-06", "CLI dispatch, init, create, board", []string{"T-02", "T-03", "T-04"}),
			golden: "task_skeleton_blocked.md",
		},
		{
			name:   "task skeleton with a title needing YAML quoting",
			got:    task.Render("T-12", "duty: wire the watcher", nil),
			golden: "task_skeleton_quoted.md",
		},
		{
			name:   "board skeleton",
			got:    board.Render("Board"),
			golden: "board_skeleton.md",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := string(tt.got), goldenFile(t, tt.golden); got != want {
				t.Errorf("%s mismatch:\ngot:\n%s\nwant:\n%s", tt.golden, got, want)
			}
		})
	}
}

func TestInitReadmeGolden(t *testing.T) {
	want := goldenFile(t, "readme.md")
	got := readText(t, filepath.Join(initDuty(t), "README.md"))
	if got != want {
		t.Errorf("README.md mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
