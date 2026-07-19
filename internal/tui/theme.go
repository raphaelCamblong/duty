package tui

import (
	"fmt"
	"image/color"
	"strconv"

	"charm.land/lipgloss/v2"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/task"
)

// adaptive is one palette slot's light and dark values (each #rrggbb or an ansi
// index 0-255); lipgloss v2 dropped the background flag, so Theme.dark resolves it.
type adaptive struct {
	Light, Dark string
}

// Theme is the TUI's semantic color palette: one adaptive slot per meaning,
// resolved once against dark. A status slot inks its word (raw hue on dark, an
// AA-readable tone on light) while its .Dark component fills that status's bar
// segments on both themes; config overrides any slot ([tui.palette]).
type Theme struct {
	// dark is the terminal mode, resolved once at startup.
	dark bool
	// Accent inks focused borders, the breadcrumb, the selection, ids, and the
	// header title.
	Accent adaptive
	// Dim inks chrome — separators, ages, hints, blurred borders.
	Dim adaptive
	// Todo inks the todo word; its .Dark hue (bronze) fills todo bar segments.
	Todo adaptive
	// InProgress inks the in-progress word; its .Dark hue (peach) fills its bars.
	InProgress adaptive
	// Done inks the done word; its .Dark hue (olive) fills done bar segments.
	Done adaptive
	// Blocked inks the blocked word on both themes, plus scan errors and drift.
	Blocked adaptive
}

func (t Theme) resolve(slot adaptive) color.Color {
	if t.dark {
		return lipgloss.Color(slot.Dark)
	}
	return lipgloss.Color(slot.Light)
}

// DefaultTheme is the frozen duty palette (§8): the raw peach/bronze/olive hues
// fill the bars on both themes and ink the status words on dark; on light those
// hues are too pale for text, so each word shifts to an AA-readable tone on white.
func DefaultTheme() Theme {
	return Theme{
		dark:       true,
		Accent:     adaptive{Light: "#1f3a5f", Dark: "#e1ebaf"},
		Dim:        adaptive{Light: "242", Dark: "243"},
		Todo:       adaptive{Light: "#8a6d00", Dark: "#af874b"},
		InProgress: adaptive{Light: "#3a6ea5", Dark: "#e1af7d"},
		Done:       adaptive{Light: "#6f7d27", Dark: "#9baf37"},
		Blocked:    adaptive{Light: "160", Dark: "203"},
	}
}

// themeFromConfig overlays the config palette onto DefaultTheme: an unset slot
// or channel keeps the default, a malformed value errors naming its key.
func themeFromConfig(palette config.Palette, dark bool) (Theme, error) {
	theme := DefaultTheme()
	theme.dark = dark
	slots := []struct {
		key string
		val *config.Color
		dst *adaptive
	}{
		{"accent", palette.Accent, &theme.Accent},
		{"dim", palette.Dim, &theme.Dim},
		{"todo", palette.Todo, &theme.Todo},
		{"in-progress", palette.InProgress, &theme.InProgress},
		{"done", palette.Done, &theme.Done},
		{"blocked", palette.Blocked, &theme.Blocked},
	}
	for _, slot := range slots {
		if slot.val == nil {
			continue
		}
		if err := overlaySlot(slot.dst, *slot.val, slot.key); err != nil {
			return Theme{}, err
		}
	}
	return theme, nil
}

func overlaySlot(dst *adaptive, value config.Color, key string) error {
	if value.Light != "" {
		if err := validColor(value.Light); err != nil {
			return fmt.Errorf("tui.palette.%s.light: %w", key, err)
		}
		dst.Light = value.Light
	}
	if value.Dark != "" {
		if err := validColor(value.Dark); err != nil {
			return fmt.Errorf("tui.palette.%s.dark: %w", key, err)
		}
		dst.Dark = value.Dark
	}
	return nil
}

func validColor(value string) error {
	if len(value) == 7 && value[0] == '#' {
		if _, err := strconv.ParseUint(value[1:], 16, 32); err == nil {
			return nil
		}
	}
	if index, err := strconv.Atoi(value); err == nil && index >= 0 && index <= 255 {
		return nil
	}
	return fmt.Errorf("invalid color %q — want #rrggbb or an ansi index 0-255", value)
}

func (t Theme) statusStyle(status string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.resolve(t.statusSlot(status)))
}

func (t Theme) statusSlot(status string) adaptive {
	switch status {
	case task.StatusInProgress:
		return t.InProgress
	case task.StatusTodo:
		return t.Todo
	case task.StatusBlocked:
		return t.Blocked
	case task.StatusDone:
		return t.Done
	}
	return t.Dim
}

// statusColor is a status's bar fill: the .Dark hue for the three flat statuses
// (on both themes), the resolved slot for blocked, dim grey otherwise.
func (t Theme) statusColor(status string) color.Color {
	switch status {
	case task.StatusInProgress:
		return lipgloss.Color(t.InProgress.Dark)
	case task.StatusTodo:
		return lipgloss.Color(t.Todo.Dark)
	case task.StatusDone:
		return lipgloss.Color(t.Done.Dark)
	case task.StatusBlocked:
		return t.resolve(t.Blocked)
	}
	return t.resolve(t.Dim)
}

func (t Theme) accent() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.resolve(t.Accent)) }

func (t Theme) dim() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.resolve(t.Dim)) }

func (t Theme) section() lipgloss.Style { return t.dim().Bold(true) }

func (t Theme) crumb() lipgloss.Style { return t.accent().Bold(true) }

func (t Theme) alert() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.resolve(t.Blocked)) }

func (t Theme) focusBox() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.resolve(t.Accent)).Padding(0, 1)
}

func (t Theme) blurBox() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.resolve(t.Dim)).Padding(0, 1)
}

func (t Theme) panelBox(focused bool) lipgloss.Style {
	if focused {
		return t.focusBox()
	}
	return t.blurBox()
}

func (t Theme) cursorMark(selected bool) string {
	if selected {
		return t.accent().Bold(true).Render("❯") + " "
	}
	return "  "
}
