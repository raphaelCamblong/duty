package tui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Run finds the tree containing cwd, loads config, and runs the live board
// viewer full-screen until quit.
func Run(cwd string) error {
	root, err := tree.FindRoot(cwd)
	if err != nil {
		return err
	}
	userPath, _ := config.UserPath()
	cfg, err := config.Load(userPath, filepath.Join(root, tree.ConfigFile))
	if err != nil {
		return err
	}
	applyTheme(cfg.TUI.Theme)
	m, err := New(root, cfg)
	if err != nil {
		return err
	}
	defer m.Close()
	w, err := NewWatcher(root)
	if err != nil {
		return err
	}
	defer w.Close()
	m.refresh = w.C
	_, err = tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

// applyTheme forces lipgloss's light/dark background flag when config says
// so; "auto" keeps terminal detection, so AdaptiveColors resolve themselves.
func applyTheme(theme string) {
	switch theme {
	case "dark":
		lipgloss.SetHasDarkBackground(true)
	case "light":
		lipgloss.SetHasDarkBackground(false)
	}
}
