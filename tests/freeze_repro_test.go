package tests

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/tui"
)

// allArchivedModel builds a model over a tree whose every task is archived.
func allArchivedModel(t *testing.T) tui.Model {
	t.Helper()
	dir := t.TempDir()
	runDuty(t, dir, "init", "x")
	tree := dir + "/duty"
	runDuty(t, tree, "create", "track", "sub")
	runDuty(t, tree, "create", "task", "one", "--in", "sub")
	runDuty(t, tree, "status", "T-01", "done")
	runDuty(t, tree, "archive")

	m, err := tui.New(fsys.OS{}, tree, config.Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	return nm.(tui.Model)
}

// TestEmptyBoardFilterIsInert pins the v0.4.0 trap: on a board with nothing
// selectable, "/" must not engage the (invisible) filter, and "q" must still
// quit afterwards.
func TestEmptyBoardFilterIsInert(t *testing.T) {
	m := allArchivedModel(t)

	nm, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = nm.(tui.Model)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatalf("q after / did not produce a command — filter swallowed it")
	}
	if msg := cmd(); msg != nil {
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Fatalf("q after / produced %T, want tea.QuitMsg", msg)
		}
	}
}

// TestAllArchivedHintNamesTheToggle pins the empty-state hint: an
// archived-out board points at "a" instead of claiming the tree is empty.
func TestAllArchivedHintNamesTheToggle(t *testing.T) {
	m := allArchivedModel(t)
	frame := m.View().Content
	if !strings.Contains(frame, "press a to browse") {
		t.Fatalf("all-archived frame missing the toggle hint:\n%s", frame)
	}
	if strings.Contains(frame, "empty tree") {
		t.Fatalf("all-archived frame still claims an empty tree:\n%s", frame)
	}
}

// TestDarkFromEnv pins the environment-based theme heuristic that replaced
// the input-eating OSC query.
func TestDarkFromEnv(t *testing.T) {
	cases := map[string]bool{
		"":       true,
		"15;0":   true,
		"0;15":   false,
		"12;8":   true,
		"0;7":    false,
		"weird":  true,
		"15;255": false,
	}
	for in, want := range cases {
		if got := tui.DarkFromEnv(in); got != want {
			t.Errorf("DarkFromEnv(%q) = %v, want %v", in, got, want)
		}
	}
}
