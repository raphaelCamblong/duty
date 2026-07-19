package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/tree"
	"github.com/raphaelCamblong/duty/internal/watch"
)

// Run finds the tree containing cwd, loads config, and runs the live board
// viewer full-screen until quit.
func Run(filesystem fsys.FS, cwd string) error {
	root, err := tree.FindRoot(filesystem, cwd)
	if err != nil {
		return err
	}
	userPath, _ := config.UserPath()
	cfg, err := config.Load(filesystem, userPath, filepath.Join(root, names.ConfigFile))
	if err != nil {
		return err
	}
	cfg.TUI.Theme = resolveTheme(cfg.TUI.Theme)
	model, err := New(filesystem, root, cfg)
	if err != nil {
		return err
	}
	defer model.Close()
	watcher, err := watch.NewWatcher(filesystem, root)
	if err != nil {
		return err
	}
	defer func() { _ = watcher.Close() }()
	model.refresh = watcher.C
	_, err = tea.NewProgram(model).Run()
	return err
}

// resolveTheme returns a concrete theme ("dark" or "light"). The "auto" case
// reads the environment instead of querying the terminal: an OSC query eats
// keystrokes as its "response" under terminals that never answer, wedging
// startup (the v0.4.0 freeze). Bubble Tea v2 carries no global background flag —
// the model resolves the palette from this mode.
func resolveTheme(theme string) string {
	dark := true
	switch theme {
	case "dark":
	case "light":
		dark = false
	default:
		dark = DarkFromEnv(os.Getenv("COLORFGBG"))
	}
	if dark {
		return "dark"
	}
	return "light"
}

// DarkFromEnv reads a COLORFGBG value ("fg;bg", bg 0-6 or 8 = dark); empty or
// garbled values default to dark, the overwhelming terminal norm.
func DarkFromEnv(colorfgbg string) bool {
	parts := strings.Split(colorfgbg, ";")
	bg, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return true
	}
	return bg <= 6 || bg == 8
}
