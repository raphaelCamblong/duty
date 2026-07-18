package tui

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone"

	"github.com/raphaelCamblong/duty/internal/humanize"
)

// entry is one line of the left panel: a sub-track, a task row, or a bare
// section header (never selected, never clickable). archived marks a task row
// that lives in a board's archive/ — rendered dim, previewable, never edited.
type entry struct {
	track    *Sub
	task     *Row
	section  string
	archived bool
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
// visible sub-track rows, then every section header followed by its task rows,
// and — when showArchive is on — a dim "Archived (N)" section of the board's
// archived tasks. With statusSort on the rows within each section are
// status-grouped for display (§8); off, they keep the board's file order.
func boardEntries(b Board, statusSort, showArchive bool) []list.Item {
	var items []list.Item
	subs := visibleSubs(b.Subs, showArchive)
	if len(subs) > 0 {
		items = append(items, entry{section: tracksSection})
	}
	for _, i := range subs {
		items = append(items, entry{track: &b.Subs[i]})
	}
	for si := range b.Sections {
		items = append(items, entry{section: b.Sections[si].Name})
		rows := b.Sections[si].Rows
		if statusSort {
			rows = sortedRows(rows)
		}
		for ri := range rows {
			items = append(items, entry{task: &rows[ri]})
		}
	}
	if showArchive && len(b.Archived) > 0 {
		items = append(items, entry{section: fmt.Sprintf("Archived (%d)", len(b.Archived))})
		for ri := range b.Archived {
			items = append(items, entry{task: &b.Archived[ri], archived: true})
		}
	}
	return items
}

// visibleSubs returns the indices of the sub-tracks the current view shows:
// all of them when the archive is on, otherwise every one except a track
// emptied by archiving (zero open tasks, at least one archived) — an
// archived-out track hides until the archive is revealed.
func visibleSubs(subs []Sub, showArchive bool) []int {
	var out []int
	for i := range subs {
		if archivedOut(subs[i], showArchive) {
			continue
		}
		out = append(out, i)
	}
	return out
}

// archivedOut reports whether a track is hidden by the archive-off rule: its
// subtree holds no open tasks but at least one archived one. A track that
// never held a task (zero archived) stays visible as an intentional container.
func archivedOut(s Sub, showArchive bool) bool {
	return !showArchive && emptiedByArchiving(s)
}

// emptiedByArchiving reports whether a track's subtree holds no open tasks but
// at least one archived one — the archived-out state, hidden while the archive
// is off and shown dim with its count while it is on.
func emptiedByArchiving(s Sub) bool {
	return s.Total == 0 && s.Archived > 0
}

// sortedRows groups a section's rows by status for display: stable by
// rollupOrder rank (unknown statuses last) so the board's build order survives
// as the tiebreak. It copies first, leaving the snapshot's board order intact
// for the toggle back.
func sortedRows(rows []Row) []Row {
	out := make([]Row, len(rows))
	copy(out, rows)
	sort.SliceStable(out, func(i, j int) bool {
		return statusRank(out[i].Status) < statusRank(out[j].Status)
	})
	return out
}

// statusRank is a status's position in rollupOrder, the display-sort ranking;
// an unknown status sorts after every known one.
func statusRank(status string) int {
	for i, st := range rollupOrder {
		if st == status {
			return i
		}
	}
	return len(rollupOrder)
}

// compactDelegate renders entries one line each: tracks with a colored
// per-status rollup, tasks with status/gates/drift columns, section names
// dim; selectable lines are wrapped in BubbleZone hit-zones.
type compactDelegate struct {
	theme       Theme
	zones       *zone.Manager
	nameW       int
	countW      int
	idW         int
	driftW      int
	ageW        int
	showAge     bool
	showGates   bool
	showArchive bool
	now         time.Time
	glyph       string
}

// newDelegate sizes a compact delegate's columns for one board; showAge governs
// the always-on relative-age column (whose width is measured here) and showGates
// the gate column that a narrow terminal drops. showArchive folds the board's
// archived rows into the id and age widths so they align with the open rows.
// glyph is the current spinner frame drawn beside in-progress rows, "" when the
// tree holds none.
func newDelegate(theme Theme, z *zone.Manager, b Board, showAge, showGates, showArchive bool, now time.Time, glyph string) compactDelegate {
	d := compactDelegate{
		theme:       theme,
		zones:       z,
		nameW:       maxSubNameWidth(b.Subs),
		countW:      maxSubCountWidth(b.Subs),
		idW:         maxIDWidth(b.Sections),
		driftW:      maxDriftWidth(b.Sections),
		showAge:     showAge,
		showGates:   showGates,
		showArchive: showArchive,
		now:         now,
		glyph:       glyph,
	}
	if showAge {
		d.ageW = maxAgeWidth(b.Sections, now)
	}
	if showArchive {
		for _, r := range b.Archived {
			d.idW = max(d.idW, lipgloss.Width(r.ID))
			if showAge {
				d.ageW = max(d.ageW, lipgloss.Width(ageCell(r, now)))
			}
		}
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
		fmt.Fprint(w, " "+d.theme.section().Render(e.section))
		return
	}
	sel := index == m.Index()
	var line string
	switch {
	case e.track != nil:
		head, tail := splitMatches(m.MatchesForItem(index), len(e.track.Name))
		line = d.trackLine(*e.track, sel, m.Width(), head, tail)
	case e.archived:
		head, tail := splitMatches(m.MatchesForItem(index), len(e.task.ID))
		line = d.archivedLine(*e.task, sel, m.Width(), head, tail)
	default:
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

// trackLine renders one sub-track: name and title left, a flexible gap, then a
// right-aligned fixed-width status-distribution bar of its subtree with a dim
// total count flush at the line end (dim "empty" when the subtree holds no
// tasks). The bar column starts at the same x on every track row — mirroring
// taskLine's right columns — and the title ellipsis-truncates first when narrow.
func (d compactDelegate) trackLine(s Sub, selected bool, w int, nameM, titleM []int) string {
	rightW := trackRightWidth(d.countW)
	fixed := 2 + d.nameW + 2 + 2 + rightW
	nameStyle := boldWhen(d.theme.accent(), selected)
	rightCell := d.theme.trackBarCell(s.Counts, d.countW)
	if d.showArchive && emptiedByArchiving(s) {
		nameStyle = boldWhen(d.theme.dim(), selected)
		rightCell = pad(d.theme.dim().Render(fmt.Sprintf("%d archived", s.Archived)), rightW)
	}
	line := d.theme.cursorMark(selected) +
		pad(styleMatches(s.Name, nameM, nameStyle), d.nameW) + "  " +
		pad(styleMatches(s.Title, titleM, boldWhen(lipgloss.NewStyle(), selected)), max(w-fixed, minTitleWidth)) + "  " +
		rightCell
	return ansi.Truncate(line, w, "…")
}

// archivedLine renders one archived task row: dim id, title, and relative age
// only — no status or gate columns, since an archived task is done by
// convention. The row stays selectable so enter opens its read-only preview.
func (d compactDelegate) archivedLine(r Row, selected bool, w int, idM, titleM []int) string {
	fixed := 2 + d.idW + 2
	if d.showAge {
		fixed += 2 + d.ageW
	}
	dim := boldWhen(d.theme.dim(), selected)
	line := d.theme.cursorMark(selected) +
		pad(styleMatches(r.ID, idM, dim), d.idW) + "  " +
		pad(styleMatches(r.Title, titleM, dim), max(w-fixed, minTitleWidth))
	if d.showAge {
		line += "  " + dim.Render(pad(ageCell(r, d.now), d.ageW))
	}
	return ansi.Truncate(line, w, "…")
}

// boldWhen adds bold to base when the row is selected, so the whole focused line
// reads bold — the chevron alone is thin feedback.
func boldWhen(base lipgloss.Style, selected bool) lipgloss.Style {
	if selected {
		return base.Bold(true)
	}
	return base
}

// taskLine renders one task: id, title, colored status, gate progress, drift
// badge. The board's widest badge yields title room so badges stay aligned; a
// narrow terminal drops the gate column so the always-on age column keeps room.
func (d compactDelegate) taskLine(r Row, selected bool, w int, idM, titleM []int) string {
	fixed := 2 + d.idW + 2 + 2 + statusColWidth
	if d.showGates {
		fixed += 2 + gatesColWidth
	}
	if d.showAge {
		fixed += 2 + d.ageW
	}
	if d.driftW > 0 {
		fixed += 2 + d.driftW
	}
	dim := boldWhen(d.theme.dim(), selected)
	line := d.theme.cursorMark(selected) +
		pad(styleMatches(r.ID, idM, boldWhen(d.theme.accent(), selected)), d.idW) + "  " +
		pad(styleMatches(r.Title, titleM, boldWhen(lipgloss.NewStyle(), selected)), max(w-fixed, minTitleWidth)) + "  " +
		d.statusCell(r, selected)
	if who := claimerTag(r); who != "" {
		line += dim.Render(" · " + who)
	}
	if wait := waitsTag(r); wait != "" {
		line += "  " + dim.Render(wait)
	}
	if d.showGates {
		line += "  " + dim.Render(pad(gatesCell(r), gatesColWidth))
	}
	if d.showAge {
		line += "  " + dim.Render(pad(ageCell(r, d.now), d.ageW))
	}
	if r.Drift != "" {
		line += "  " + boldWhen(d.theme.alert(), selected).Render(pad("⚠ "+r.Drift, d.driftW))
	}
	return ansi.Truncate(line, w, "…")
}

// statusCell renders a row's status word in its status color, the spinner glyph
// trailing beside an in-progress status so the fixed status column stays aligned
// across rows.
func (d compactDelegate) statusCell(r Row, selected bool) string {
	cell := boldWhen(d.theme.statusStyle(r.Status), selected).Render(pad(r.Status, statusColWidth))
	if inProgress(r) && d.glyph != "" {
		cell += " " + d.glyph
	}
	return cell
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
