package tui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/tree"
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
	w, err := NewWatcher(f, root)
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()
	m.refresh = w.C
	_, err = tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

// resolveTheme pins lipgloss's background flag and returns a concrete glamour
// style ("dark" or "light"). The "auto" case runs the terminal-background
// query exactly once, here, before the program starts — so neither
// AdaptiveColors nor the markdown renderer ever query mid-frame.
func resolveTheme(theme string) string {
	switch theme {
	case "dark":
		lipgloss.SetHasDarkBackground(true)
		return "dark"
	case "light":
		lipgloss.SetHasDarkBackground(false)
		return "light"
	default:
		dark := lipgloss.HasDarkBackground()
		lipgloss.SetHasDarkBackground(dark)
		if dark {
			return "dark"
		}
		return "light"
	}
}
