package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone"
)

// entry is one line of the left panel: a sub-track, a task row, or a bare
// section header (never selected, never clickable).
type entry struct {
	track   *Sub
	task    *Row
	section string
}

// FilterValue feeds the list's fuzzy filter: track name+title for tracks,
// id+title for tasks, nothing for section headers so they filter away.
func (e entry) FilterValue() string {
	switch {
	case e.track != nil:
		return e.track.Name + " " + e.track.Title
	case e.task != nil:
		return e.task.ID + " " + e.task.Title
	}
	return ""
}

// selectable reports whether the entry is a track or a task, not a header.
func (e entry) selectable() bool {
	return e.track != nil || e.task != nil
}

// boardEntries lists a board as left-panel entries: sub-tracks first, then
// every section header followed by its task rows in board order.
func boardEntries(b Board) []list.Item {
	var items []list.Item
	for i := range b.Subs {
		items = append(items, entry{track: &b.Subs[i]})
	}
	for si := range b.Sections {
		items = append(items, entry{section: b.Sections[si].Name})
		for ri := range b.Sections[si].Rows {
			items = append(items, entry{task: &b.Sections[si].Rows[ri]})
		}
	}
	return items
}

// compactDelegate renders entries one line each: tracks with a colored
// per-status rollup, tasks with status/gates/drift columns, section names
// dim; selectable lines are wrapped in BubbleZone hit-zones.
type compactDelegate struct {
	zones  *zone.Manager
	nameW  int
	idW    int
	driftW int
}

// newDelegate sizes a compact delegate's columns for one board.
func newDelegate(z *zone.Manager, b Board) compactDelegate {
	return compactDelegate{
		zones:  z,
		nameW:  maxSubNameWidth(b.Subs),
		idW:    maxIDWidth(b.Sections),
		driftW: maxDriftWidth(b.Sections),
	}
}

// Height is one line per entry, satisfying list.ItemDelegate.
func (d compactDelegate) Height() int { return 1 }

// Spacing is zero lines between entries, satisfying list.ItemDelegate.
func (d compactDelegate) Spacing() int { return 0 }

// Update ignores messages, satisfying list.ItemDelegate.
func (d compactDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

// Render writes one entry's line, underlining fuzzy-filter matches.
func (d compactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	e, ok := item.(entry)
	if !ok {
		return
	}
	if e.section != "" {
		fmt.Fprint(w, " "+sectionStyle.Render(e.section))
		return
	}
	sel := index == m.Index()
	var line string
	if e.track != nil {
		head, tail := splitMatches(m.MatchesForItem(index), len(e.track.Name))
		line = d.trackLine(*e.track, sel, m.Width(), head, tail)
	} else {
		head, tail := splitMatches(m.MatchesForItem(index), len(e.task.ID))
		line = d.taskLine(*e.task, sel, m.Width(), head, tail)
	}
	fmt.Fprint(w, d.zones.Mark(itemZone(index), line))
}

// itemZone is the stable zone name for the visible entry at index.
func itemZone(index int) string {
	return fmt.Sprintf("item-%d", index)
}

// splitMatches splits filter-match rune indices around a head of headLen
// runes followed by one separator, re-basing the tail indices.
func splitMatches(matches []int, headLen int) (head, tail []int) {
	for _, i := range matches {
		if i < headLen {
			head = append(head, i)
			continue
		}
		if i > headLen {
			tail = append(tail, i-headLen-1)
		}
	}
	return head, tail
}

// styleMatches renders s in base with matched runes underlined.
func styleMatches(s string, matches []int, base lipgloss.Style) string {
	if len(matches) == 0 {
		return base.Render(s)
	}
	return lipgloss.StyleRunes(s, matches, base.Underline(true), base)
}

// trackLine renders one sub-track: name, title, and a per-status rollup of
// its subtree, each count in its status color, zero-count statuses omitted.
func (d compactDelegate) trackLine(s Sub, selected bool, w int, nameM, titleM []int) string {
	title := lipgloss.NewStyle()
	if selected {
		title = selStyle
	}
	line := cursorMark(selected) +
		pad(styleMatches(s.Name, nameM, accentStyle), d.nameW) + "  " +
		styleMatches(s.Title, titleM, title) + "  " +
		rollupOrEmpty(s.Counts)
	return ansi.Truncate(line, w, "…")
}

// taskLine renders one task: id, title, colored status, gate progress, drift
// badge. The board's widest badge yields title room so badges stay aligned.
func (d compactDelegate) taskLine(r Row, selected bool, w int, idM, titleM []int) string {
	fixed := 2 + d.idW + 2 + 2 + statusColWidth + 2 + gatesColWidth
	if d.driftW > 0 {
		fixed += 2 + d.driftW
	}
	title := lipgloss.NewStyle()
	if selected {
		title = selStyle
	}
	line := cursorMark(selected) +
		pad(styleMatches(r.ID, idM, accentStyle), d.idW) + "  " +
		pad(styleMatches(r.Title, titleM, title), max(w-fixed, minTitleWidth)) + "  " +
		statusStyle(r.Status).Render(pad(r.Status, statusColWidth)) + "  " +
		dimStyle.Render(pad(gatesCell(r), gatesColWidth))
	if r.Drift != "" {
		line += "  " + driftStyle.Render(pad("⚠ "+r.Drift, d.driftW))
	}
	return ansi.Truncate(line, w, "…")
}
