package tui

import (
	"fmt"
	"io"
	"sort"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/raphaelCamblong/duty/internal/humanize"
)

type entry struct {
	track    *Sub
	task     *Row
	section  string
	archived bool
}

func (e entry) FilterValue() string {
	switch {
	case e.track != nil:
		return e.track.Name + " " + e.track.Title
	case e.task != nil:
		return e.task.ID + " " + e.task.Title
	}
	return ""
}

func (e entry) selectable() bool {
	return e.track != nil || e.task != nil
}

func anySelectable(items []list.Item) bool {
	for _, it := range items {
		if item, ok := it.(entry); ok && item.selectable() {
			return true
		}
	}
	return false
}

const tracksSection = "Tracks"

// boardEntries lists a board as left-panel entries: a "Tracks" header over the
// visible sub-track rows, then every section header followed by its task rows,
// and — when showArchive is on — a dim "Archived (N)" section of the board's
// archived tasks. With statusSort on the rows within each section are
// status-grouped for display (§8); off, they keep the board's file order.
func boardEntries(board Board, statusSort, showArchive bool) []list.Item {
	var items []list.Item
	subs := visibleSubs(board.Subs, showArchive)
	if len(subs) > 0 {
		items = append(items, entry{section: tracksSection})
	}
	for _, i := range subs {
		items = append(items, entry{track: &board.Subs[i]})
	}
	for si := range board.Sections {
		items = append(items, entry{section: board.Sections[si].Name})
		rows := board.Sections[si].Rows
		if statusSort {
			rows = sortedRows(rows)
		}
		for ri := range rows {
			items = append(items, entry{task: &rows[ri]})
		}
	}
	if showArchive && len(board.Archived) > 0 {
		items = append(items, entry{section: fmt.Sprintf("Archived (%d)", len(board.Archived))})
		for ri := range board.Archived {
			items = append(items, entry{task: &board.Archived[ri], archived: true})
		}
	}
	return items
}

func visibleSubs(subs []Sub, showArchive bool) []int {
	var out []int
	for i := range subs {
		if !showArchive && emptiedByArchiving(subs[i]) {
			continue
		}
		out = append(out, i)
	}
	return out
}

func emptiedByArchiving(sub Sub) bool {
	return sub.Total == 0 && sub.Archived > 0
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

func statusRank(status string) int {
	for i, st := range rollupOrder {
		if st == status {
			return i
		}
	}
	return len(rollupOrder)
}

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

// viewOpts is how a board renders right now: which optional columns show, the
// clock for relative ages, and the spinner glyph for in-progress rows.
type viewOpts struct {
	showAge     bool
	showGates   bool
	showArchive bool
	now         time.Time
	glyph       string
}

func newDelegate(theme Theme, zones *zone.Manager, board Board, opts viewOpts) compactDelegate {
	delegate := compactDelegate{
		theme:       theme,
		zones:       zones,
		nameW:       maxSubNameWidth(board.Subs),
		countW:      maxSubCountWidth(board.Subs),
		idW:         maxIDWidth(board.Sections),
		driftW:      maxDriftWidth(board.Sections),
		showAge:     opts.showAge,
		showGates:   opts.showGates,
		showArchive: opts.showArchive,
		now:         opts.now,
		glyph:       opts.glyph,
	}
	if opts.showAge {
		delegate.ageW = maxAgeWidth(board.Sections, opts.now)
	}
	if opts.showArchive {
		delegate = widenForArchive(delegate, board.Archived, opts)
	}
	return delegate
}

// widenForArchive grows the id and age columns to fit the board's archived
// rows, which the sections' widths don't cover.
func widenForArchive(delegate compactDelegate, archived []Row, opts viewOpts) compactDelegate {
	for _, row := range archived {
		delegate.idW = max(delegate.idW, lipgloss.Width(row.ID))
		if opts.showAge {
			delegate.ageW = max(delegate.ageW, lipgloss.Width(ageCell(row, opts.now)))
		}
	}
	return delegate
}

// Height is one line per entry, satisfying list.ItemDelegate.
func (d compactDelegate) Height() int { return 1 }

// Spacing is zero lines between entries, satisfying list.ItemDelegate.
func (d compactDelegate) Spacing() int { return 0 }

// Update ignores messages, satisfying list.ItemDelegate.
func (d compactDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

// Render writes one entry's line, underlining fuzzy-filter matches.
func (d compactDelegate) Render(writer io.Writer, listModel list.Model, index int, item list.Item) {
	current, ok := item.(entry)
	if !ok {
		return
	}
	if current.section != "" {
		fmt.Fprint(writer, " "+d.theme.section().Render(current.section))
		return
	}
	sel := index == listModel.Index()
	var line string
	switch {
	case current.track != nil:
		line = d.trackLine(*current.track, sel, listModel.Width(), splitMatches(listModel.MatchesForItem(index), len(current.track.Name)))
	case current.archived:
		line = d.archivedLine(*current.task, sel, listModel.Width(), splitMatches(listModel.MatchesForItem(index), len(current.task.ID)))
	default:
		line = d.taskLine(*current.task, sel, listModel.Width(), splitMatches(listModel.MatchesForItem(index), len(current.task.ID)))
	}
	fmt.Fprint(writer, d.zones.Mark(itemZone(index), line))
}

func itemZone(index int) string {
	return fmt.Sprintf("item-%d", index)
}

// matches is one row's filter-match rune indices split across its two styled
// cells: head into the id/name cell, tail into the title cell (re-based).
type matches struct {
	head []int
	tail []int
}

// splitMatches splits filter-match rune indices around a head of headLen
// runes followed by one separator, re-basing the tail indices.
func splitMatches(all []int, headLen int) matches {
	var result matches
	for _, index := range all {
		if index < headLen {
			result.head = append(result.head, index)
			continue
		}
		if index > headLen {
			result.tail = append(result.tail, index-headLen-1)
		}
	}
	return result
}

func styleMatches(text string, matches []int, base lipgloss.Style) string {
	if len(matches) == 0 {
		return base.Render(text)
	}
	return lipgloss.StyleRunes(text, matches, base.Underline(true), base)
}

// trackLine renders one sub-track: name and title left, a flexible gap, then a
// right-aligned fixed-width status-distribution bar of its subtree with a dim
// total count flush at the line end (dim "empty" when the subtree holds no
// tasks). The bar column starts at the same x on every track row — mirroring
// taskLine's right columns — and the title ellipsis-truncates first when narrow.
func (d compactDelegate) trackLine(sub Sub, selected bool, width int, match matches) string {
	rightW := trackRightWidth(d.countW)
	fixed := 2 + d.nameW + 2 + 2 + rightW
	nameStyle := boldWhen(d.theme.accent(), selected)
	rightCell := d.theme.trackBarCell(sub.Counts, d.countW)
	if d.showArchive && emptiedByArchiving(sub) {
		nameStyle = boldWhen(d.theme.dim(), selected)
		rightCell = pad(d.theme.dim().Render(fmt.Sprintf("%d archived", sub.Archived)), rightW)
	}
	line := d.theme.cursorMark(selected) +
		pad(styleMatches(sub.Name, match.head, nameStyle), d.nameW) + "  " +
		pad(styleMatches(sub.Title, match.tail, boldWhen(lipgloss.NewStyle(), selected)), max(width-fixed, minTitleWidth)) + "  " +
		rightCell
	return ansi.Truncate(line, width, "…")
}

// archivedLine renders one archived task row: dim id, title, and relative age
// only — no status or gate columns, since an archived task is done by
// convention. The row stays selectable so enter opens its read-only preview.
func (d compactDelegate) archivedLine(row Row, selected bool, width int, match matches) string {
	fixed := 2 + d.idW + 2
	if d.showAge {
		fixed += 2 + d.ageW
	}
	dim := boldWhen(d.theme.dim(), selected)
	line := d.theme.cursorMark(selected) +
		pad(styleMatches(row.ID, match.head, dim), d.idW) + "  " +
		pad(styleMatches(row.Title, match.tail, dim), max(width-fixed, minTitleWidth))
	if d.showAge {
		line += "  " + dim.Render(pad(ageCell(row, d.now), d.ageW))
	}
	return ansi.Truncate(line, width, "…")
}

func boldWhen(base lipgloss.Style, selected bool) lipgloss.Style {
	if selected {
		return base.Bold(true)
	}
	return base
}

// taskLine renders one task: id, title, colored status, gate progress, drift
// badge. The board's widest badge yields title room so badges stay aligned; a
// narrow terminal drops the gate column so the always-on age column keeps room.
func (d compactDelegate) taskLine(row Row, selected bool, width int, match matches) string {
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
		pad(styleMatches(row.ID, match.head, boldWhen(d.theme.accent(), selected)), d.idW) + "  " +
		pad(styleMatches(row.Title, match.tail, boldWhen(lipgloss.NewStyle(), selected)), max(width-fixed, minTitleWidth)) + "  " +
		d.statusCell(row, selected)
	if who := claimerTag(row); who != "" {
		line += dim.Render(" · " + who)
	}
	if wait := waitsTag(row); wait != "" {
		line += "  " + dim.Render(wait)
	}
	if d.showGates {
		line += "  " + dim.Render(pad(gatesCell(row), gatesColWidth))
	}
	if d.showAge {
		line += "  " + dim.Render(pad(ageCell(row, d.now), d.ageW))
	}
	if row.DriftText != "" {
		line += "  " + boldWhen(d.theme.alert(), selected).Render(pad("⚠ "+row.DriftText, d.driftW))
	}
	return ansi.Truncate(line, width, "…")
}

func (d compactDelegate) statusCell(row Row, selected bool) string {
	cell := boldWhen(d.theme.statusStyle(row.Status), selected).Render(pad(row.Status, statusColWidth))
	if inProgress(row) && d.glyph != "" {
		cell += " " + d.glyph
	}
	return cell
}

func ageCell(row Row, now time.Time) string {
	if row.UpdatedAt.IsZero() {
		return ""
	}
	return humanize.RelTime(row.UpdatedAt, now)
}

func maxAgeWidth(sections []Section, now time.Time) int {
	width := 0
	for _, section := range sections {
		for _, row := range section.Rows {
			width = max(width, lipgloss.Width(ageCell(row, now)))
		}
	}
	return width
}
