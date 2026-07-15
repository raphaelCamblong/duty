package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap names every binding of the viewer, with help text ready for a
// bubbles/help footer. It satisfies help.KeyMap.
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Back    key.Binding
	Focus   key.Binding
	Filter  key.Binding
	Edit    key.Binding
	Age     key.Binding
	Sort    key.Binding
	Refresh key.Binding
	Help    key.Binding
	Quit    key.Binding
}

// defaultKeys returns the spec §8 bindings.
func defaultKeys() keyMap {
	return keyMap{
		Up:      key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down:    key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "open")),
		Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Focus:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "panel")),
		Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Age:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "age")),
		Sort:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "keys")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ShortHelp is the one-line hint bar shown by default.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open, k.Back, k.Focus, k.Filter, k.Edit, k.Refresh, k.Help, k.Quit}
}

// FullHelp is the expanded grid shown after "?".
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Open, k.Back, k.Focus},
		{k.Filter, k.Edit, k.Age, k.Sort, k.Refresh},
		{k.Help, k.Quit},
	}
}
