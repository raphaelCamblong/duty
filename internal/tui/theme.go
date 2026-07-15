package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/lipgloss"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/task"
)

// Theme is the TUI's semantic color palette: one adaptive slot per meaning.
// Each status slot inks its word directly — the raw duty hue on dark, an
// AA-readable tone on light — while its .Dark component fills that status's
// distribution-bar segments on both themes. Accent inks focused chrome and
// ids, Dim inks separators and ages, Blocked inks the blocked word plus scan
// errors and drift. DefaultTheme is the frozen default (§8); config overrides
// any slot ([tui.palette]).
type Theme struct {
	// Accent inks focused borders, the breadcrumb, the selection, ids, and the
	// header title.
	Accent lipgloss.AdaptiveColor
	// Dim inks chrome — separators, ages, hints, blurred borders.
	Dim lipgloss.AdaptiveColor
	// Todo inks the todo word; its .Dark hue (bronze) fills todo bar segments.
	Todo lipgloss.AdaptiveColor
	// InProgress inks the in-progress word; its .Dark hue (peach) fills its bars.
	InProgress lipgloss.AdaptiveColor
	// Done inks the done word; its .Dark hue (olive) fills done bar segments.
	Done lipgloss.AdaptiveColor
	// Blocked inks the blocked word on both themes, plus scan errors and drift.
	Blocked lipgloss.AdaptiveColor
}

// DefaultTheme is the frozen duty palette (§8): peach #e1af7d, bronze #af874b,
// olive #9baf37 fill the distribution bars on both themes and, on dark
// terminals, ink the status words directly. On light terminals those hues are
// too pale for ink (peach 1.9, olive 2.3 to 1 on white), so each word shifts to
// a flat AA-readable tone measured on white: in-progress blue #3a6ea5 (5.3:1),
// todo amber #8a6d00 (4.9:1), done olive #6f7d27 (4.5:1), accent navy #1f3a5f
// (11.5:1). blocked stays red on both — the palette carries no alarm color.
func DefaultTheme() Theme {
	return Theme{
		Accent:     lipgloss.AdaptiveColor{Light: "#1f3a5f", Dark: "#e1ebaf"},
		Dim:        lipgloss.AdaptiveColor{Light: "242", Dark: "243"},
		Todo:       lipgloss.AdaptiveColor{Light: "#8a6d00", Dark: "#af874b"},
		InProgress: lipgloss.AdaptiveColor{Light: "#3a6ea5", Dark: "#e1af7d"},
		Done:       lipgloss.AdaptiveColor{Light: "#6f7d27", Dark: "#9baf37"},
		Blocked:    lipgloss.AdaptiveColor{Light: "160", Dark: "203"},
	}
}

// themeFromConfig overlays the config palette onto DefaultTheme: an unset slot
// (and, in the table form, an unset channel) keeps the default; a malformed
// value errors naming its key.
func themeFromConfig(p config.Palette) (Theme, error) {
	t := DefaultTheme()
	slots := []struct {
		key string
		val *config.Color
		dst *lipgloss.AdaptiveColor
	}{
		{"accent", p.Accent, &t.Accent},
		{"dim", p.Dim, &t.Dim},
		{"todo", p.Todo, &t.Todo},
		{"in-progress", p.InProgress, &t.InProgress},
		{"done", p.Done, &t.Done},
		{"blocked", p.Blocked, &t.Blocked},
	}
	for _, s := range slots {
		if s.val == nil {
			continue
		}
		if err := overlaySlot(s.dst, *s.val, s.key); err != nil {
			return Theme{}, err
		}
	}
	return t, nil
}

// overlaySlot validates the light and dark channels of c and writes each set
// one onto dst, leaving an empty channel at its default.
func overlaySlot(dst *lipgloss.AdaptiveColor, c config.Color, key string) error {
	if c.Light != "" {
		if err := validColor(c.Light); err != nil {
			return fmt.Errorf("tui.palette.%s.light: %w", key, err)
		}
		dst.Light = c.Light
	}
	if c.Dark != "" {
		if err := validColor(c.Dark); err != nil {
			return fmt.Errorf("tui.palette.%s.dark: %w", key, err)
		}
		dst.Dark = c.Dark
	}
	return nil
}

// validColor accepts a #rrggbb hex triplet or an ansi index 0-255 — the two
// color forms the duty palette uses.
func validColor(s string) error {
	if len(s) == 7 && s[0] == '#' {
		if _, err := strconv.ParseUint(s[1:], 16, 32); err == nil {
			return nil
		}
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n <= 255 {
		return nil
	}
	return fmt.Errorf("invalid color %q — want #rrggbb or an ansi index 0-255", s)
}

// statusStyle inks a status word as flat colored text: on dark each word keeps
// its raw palette hue, on light it shifts to an AA-readable tone; blocked is
// red, an unknown status dim.
func (t Theme) statusStyle(status string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.statusInk(status))
}

// statusInk is a status word's adaptive foreground.
func (t Theme) statusInk(status string) lipgloss.AdaptiveColor {
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

// statusColor is a status's distribution-bar fill: the status ink's .Dark hue
// for the three flat palette statuses (peach, bronze, olive on both themes),
// the full adaptive slot for blocked and for an unknown status.
func (t Theme) statusColor(status string) lipgloss.TerminalColor {
	switch status {
	case task.StatusInProgress:
		return lipgloss.Color(t.InProgress.Dark)
	case task.StatusTodo:
		return lipgloss.Color(t.Todo.Dark)
	case task.StatusDone:
		return lipgloss.Color(t.Done.Dark)
	case task.StatusBlocked:
		return t.Blocked
	}
	return t.Dim
}

// accent styles text in the accent hue (ids, breadcrumb, selection, title).
func (t Theme) accent() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.Accent) }

// dim styles chrome text — separators, ages, hints — in the dim grey.
func (t Theme) dim() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.Dim) }

// section styles a bold dim section header.
func (t Theme) section() lipgloss.Style { return t.dim().Bold(true) }

// crumb styles a bold accent breadcrumb segment.
func (t Theme) crumb() lipgloss.Style { return t.accent().Bold(true) }

// alert styles scan errors and drift badges in the blocked hue.
func (t Theme) alert() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.Blocked) }

// focusBox is the rounded accent-bordered box of a focused panel and the header.
func (t Theme) focusBox() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Accent).Padding(0, 1)
}

// blurBox is the rounded dim-bordered box of a blurred panel.
func (t Theme) blurBox() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Dim).Padding(0, 1)
}

// panelBox is a panel's border style: accent when focused, dim otherwise.
func (t Theme) panelBox(focused bool) lipgloss.Style {
	if focused {
		return t.focusBox()
	}
	return t.blurBox()
}

// cursorMark is the two-column selection gutter.
func (t Theme) cursorMark(selected bool) string {
	if selected {
		return t.accent().Bold(true).Render("❯") + " "
	}
	return "  "
}
