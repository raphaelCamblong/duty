package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/tui"
)

// themeFrame renders the deterministic 120x35 browsing frame for cfg with the
// age column hidden (ages carry wall-clock times that would defeat a golden).
// Bubble Tea v2's lipgloss renders full-color unconditionally and reads the
// dark/light mode from cfg.TUI.Theme, so the old global colour-profile pin is
// gone; dark is retained only to key the golden file names.
func themeFrame(t *testing.T, root string, cfg config.Config, dark bool) string {
	t.Helper()
	_ = dark
	m, err := tui.New(fsys.OS{}, root, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	m = nm.(tui.Model)
	m, _ = press(t, m, "t") // hide the wall-clock age column
	return m.View().Content
}

// testdataDir is the absolute path to tests/testdata, resolved before any
// fixture t.Chdir's the process into a temp tree.
func testdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Join(wd, "testdata")
}

// assertGolden compares got to dir/name, or rewrites it when UPDATE_GOLDEN is
// set in the environment.
func assertGolden(t *testing.T, dir, name, got string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", name, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	if got != string(want) {
		t.Errorf("%s mismatch (run with UPDATE_GOLDEN=1 to inspect)", name)
	}
}

func TestThemeDefaultByteIdentity(t *testing.T) {
	dir := testdataDir(t)
	root := fourStatusTree(t)
	for _, tc := range []struct {
		name   string
		dark   bool
		golden string
	}{
		{"dark", true, "tui_frame_dark.golden"},
		{"light", false, "tui_frame_light.golden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{Editor: "vi"}
			cfg.TUI.Theme = tc.name
			assertGolden(t, dir, tc.golden, themeFrame(t, root, cfg, tc.dark))
		})
	}
}

// These are the raw TrueColor foreground codes of the default dark palette;
// the override test asserts a recolored slot swaps out only its own hue.
const (
	codeDoneDark   = "\x1b[38;2;155;175;55m"  // olive #9baf37 — done ink and bar
	codeInProgDark = "\x1b[38;2;225;175;125m" // peach #e1af7d — in-progress ink and bar
	codeMagenta    = "\x1b[38;2;255;0;255m"   // #ff00ff — the override hue
)

func TestThemePaletteOverride(t *testing.T) {
	root := fourStatusTree(t)

	base := config.Config{Editor: "vi"}
	base.TUI.Theme = "dark"
	baseFrame := themeFrame(t, root, base, true)
	if !strings.Contains(baseFrame, codeDoneDark) {
		t.Fatalf("default dark frame is missing the olive done hue — fixture drift")
	}

	for _, form := range []struct {
		name  string
		color *config.Color
	}{
		{"bare string sets both channels", &config.Color{Light: "#ff00ff", Dark: "#ff00ff"}},
		{"table dark channel", &config.Color{Dark: "#ff00ff"}},
	} {
		t.Run(form.name, func(t *testing.T) {
			cfg := config.Config{Editor: "vi"}
			cfg.TUI.Theme = "dark"
			cfg.TUI.Palette.Done = form.color

			frame := themeFrame(t, root, cfg, true)
			if !strings.Contains(frame, codeMagenta) {
				t.Errorf("override did not recolor the done slot (no magenta):\n%s", frame)
			}
			if strings.Contains(frame, codeDoneDark) {
				t.Errorf("the old olive done hue survived the override:\n%s", frame)
			}
			if !strings.Contains(frame, codeInProgDark) {
				t.Errorf("overriding done disturbed the in-progress slot (peach gone):\n%s", frame)
			}
		})
	}
}

func TestBacklogRendersDim(t *testing.T) {
	root := fiveStatusTree(t)
	for _, tc := range []struct {
		name string
		dark bool
		dim  string // the dim-grey foreground the backlog word inherits
	}{
		{"dark", true, "\x1b[38;5;243m"},
		{"light", false, "\x1b[38;5;242m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{Editor: "vi"}
			cfg.TUI.Theme = tc.name
			frame := themeFrame(t, root, cfg, tc.dark)
			if !strings.Contains(frame, tc.dim+"backlog") {
				t.Errorf("backlog word not inked dim grey %q:\n%s", tc.dim, frame)
			}
		})
	}
}

func TestThemeMalformedColor(t *testing.T) {
	root := fourStatusTree(t)
	for _, tc := range []struct {
		name  string
		color *config.Color
		want  string
	}{
		{"bad hex", &config.Color{Dark: "#ggg"}, "tui.palette.blocked.dark"},
		{"ansi out of range", &config.Color{Light: "300"}, "tui.palette.blocked.light"},
		{"not a color", &config.Color{Dark: "cerulean"}, "tui.palette.blocked.dark"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{Editor: "vi"}
			cfg.TUI.Theme = "dark"
			cfg.TUI.Palette.Blocked = tc.color
			_, err := tui.New(fsys.OS{}, root, cfg)
			if err == nil {
				t.Fatalf("New() error = nil, want an error naming %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name the key %q", err, tc.want)
			}
		})
	}
}
