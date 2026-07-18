package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/tree"
	"github.com/raphaelCamblong/duty/internal/watch"
)

// Run finds the tree containing cwd, loads config, and runs the live board
// viewer full-screen until quit.
func Run(f fsys.FS, cwd string) error {
	root, err := tree.FindRoot(f, cwd)
	if err != nil {
		return err
	}
	userPath, _ := config.UserPath()
	cfg, err := config.Load(f, userPath, filepath.Join(root, names.ConfigFile))
	if err != nil {
		return err
	}
	cfg.TUI.Theme = resolveTheme(cfg.TUI.Theme)
	m, err := New(f, root, cfg)
	if err != nil {
		return err
	}
	defer m.Close()
	w, err := watch.NewWatcher(f, root)
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()
	m.refresh = w.C
	_, err = tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

// resolveTheme pins lipgloss's background flag and returns a concrete glamour
// style ("dark" or "light"). The "auto" case reads the environment instead of
// querying the terminal: an OSC query eats keystrokes as its "response" under
// terminals that never answer, wedging startup (the v0.4.0 freeze).
func resolveTheme(theme string) string {
	dark := true
	switch theme {
	case "dark":
	case "light":
		dark = false
	default:
		dark = DarkFromEnv(os.Getenv("COLORFGBG"))
	}
	lipgloss.SetHasDarkBackground(dark)
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
