package tui

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone"

	"github.com/raphaelCamblong/duty/internal/humanize"
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

// anySelectable reports whether items holds at least one track or task, so a
// board carrying only an empty section header still reads as empty.
func anySelectable(items []list.Item) bool {
	for _, it := range items {
		if e, ok := it.(entry); ok && e.selectable() {
			return true
		}
	}
	return false
}

// tracksSection is the label of the header above the sub-track rows; like a
// task section header it is non-selectable and filters away.
const tracksSection = "Tracks"

// boardEntries lists a board as left-panel entries: a "Tracks" header over the
// sub-track rows, then every section header followed by its task rows in board
// order.
func boardEntries(b Board) []list.Item {
	var items []list.Item
	if len(b.Subs) > 0 {
		items = append(items, entry{section: tracksSection})
	}
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
	zones   *zone.Manager
	nameW   int
	idW     int
	driftW  int
	ageW    int
	showAge bool
	now     time.Time
}

// newDelegate sizes a compact delegate's columns for one board; showAge and now
// govern the trailing relative-age column, whose width is measured here.
func newDelegate(z *zone.Manager, b Board, showAge bool, now time.Time) compactDelegate {
	d := compactDelegate{
		zones:   z,
		nameW:   maxSubNameWidth(b.Subs),
		idW:     maxIDWidth(b.Sections),
		driftW:  maxDriftWidth(b.Sections),
		showAge: showAge,
		now:     now,
	}
	if showAge {
		d.ageW = maxAgeWidth(b.Sections, now)
	}
	return d
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

// trackLine renders one sub-track: name, title, and a fixed-width inline
// status-distribution bar of its subtree with a dim total count (dim "empty"
// when the subtree holds no tasks).
func (d compactDelegate) trackLine(s Sub, selected bool, w int, nameM, titleM []int) string {
	line := cursorMark(selected) +
		pad(styleMatches(s.Name, nameM, accentStyle), d.nameW) + "  " +
		styleMatches(s.Title, titleM, titleStyle(selected)) + "  " +
		trackBarCell(s.Counts)
	return ansi.Truncate(line, w, "…")
}

// titleStyle is the entry title's style: bold when the row is selected, plain
// otherwise.
func titleStyle(selected bool) lipgloss.Style {
	if selected {
		return selStyle
	}
	return lipgloss.NewStyle()
}

// taskLine renders one task: id, title, colored status, gate progress, drift
// badge. The board's widest badge yields title room so badges stay aligned.
func (d compactDelegate) taskLine(r Row, selected bool, w int, idM, titleM []int) string {
	fixed := 2 + d.idW + 2 + 2 + statusColWidth + 2 + gatesColWidth
	if d.showAge {
		fixed += 2 + d.ageW
	}
	if d.driftW > 0 {
		fixed += 2 + d.driftW
	}
	line := cursorMark(selected) +
		pad(styleMatches(r.ID, idM, accentStyle), d.idW) + "  " +
		pad(styleMatches(r.Title, titleM, titleStyle(selected)), max(w-fixed, minTitleWidth)) + "  " +
		statusStyle(r.Status).Render(pad(r.Status, statusColWidth)) + "  " +
		dimStyle.Render(pad(gatesCell(r), gatesColWidth))
	if d.showAge {
		line += "  " + dimStyle.Render(pad(ageCell(r, d.now), d.ageW))
	}
	if r.Drift != "" {
		line += "  " + driftStyle.Render(pad("⚠ "+r.Drift, d.driftW))
	}
	return ansi.Truncate(line, w, "…")
}

// ageCell renders a task row's relative file age, "" when the row has no file
// (its modification time is unknown).
func ageCell(r Row, now time.Time) string {
	if r.UpdatedAt.IsZero() {
		return ""
	}
	return humanize.RelTime(r.UpdatedAt, now)
}

// maxAgeWidth sizes the relative-age column across every section.
func maxAgeWidth(sections []Section, now time.Time) int {
	w := 0
	for _, s := range sections {
		for _, r := range s.Rows {
			w = max(w, lipgloss.Width(ageCell(r, now)))
		}
	}
	return w
}
