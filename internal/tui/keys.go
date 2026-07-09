package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap names every binding of the viewer, with help text ready for a
// bubbles/help footer.
type keyMap struct {
	Up   key.Binding
	Down key.Binding
	Open key.Binding
	Back key.Binding
	Edit key.Binding
	Quit key.Binding
}

// defaultKeys returns the spec §8 bindings.
func defaultKeys() keyMap {
	return keyMap{
		Up:   key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down: key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Open: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Edit: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
