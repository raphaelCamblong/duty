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
		if e, ok := it.(entry); ok && e.selectable() {
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

func newDelegate(theme Theme, z *zone.Manager, b Board, opts viewOpts) compactDelegate {
	d := compactDelegate{
		theme:       theme,
		zones:       z,
		nameW:       maxSubNameWidth(b.Subs),
		countW:      maxSubCountWidth(b.Subs),
		idW:         maxIDWidth(b.Sections),
		driftW:      maxDriftWidth(b.Sections),
		showAge:     opts.showAge,
		showGates:   opts.showGates,
		showArchive: opts.showArchive,
		now:         opts.now,
		glyph:       opts.glyph,
	}
	if opts.showAge {
		d.ageW = maxAgeWidth(b.Sections, opts.now)
	}
	if opts.showArchive {
		for _, r := range b.Archived {
			d.idW = max(d.idW, lipgloss.Width(r.ID))
			if opts.showAge {
				d.ageW = max(d.ageW, lipgloss.Width(ageCell(r, opts.now)))
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
		line = d.trackLine(*e.track, sel, m.Width(), splitMatches(m.MatchesForItem(index), len(e.track.Name)))
	case e.archived:
		line = d.archivedLine(*e.task, sel, m.Width(), splitMatches(m.MatchesForItem(index), len(e.task.ID)))
	default:
		line = d.taskLine(*e.task, sel, m.Width(), splitMatches(m.MatchesForItem(index), len(e.task.ID)))
	}
	fmt.Fprint(w, d.zones.Mark(itemZone(index), line))
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
	var m matches
	for _, i := range all {
		if i < headLen {
			m.head = append(m.head, i)
			continue
		}
		if i > headLen {
			m.tail = append(m.tail, i-headLen-1)
		}
	}
	return m
}

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
func (d compactDelegate) trackLine(s Sub, selected bool, w int, m matches) string {
	rightW := trackRightWidth(d.countW)
	fixed := 2 + d.nameW + 2 + 2 + rightW
	nameStyle := boldWhen(d.theme.accent(), selected)
	rightCell := d.theme.trackBarCell(s.Counts, d.countW)
	if d.showArchive && emptiedByArchiving(s) {
		nameStyle = boldWhen(d.theme.dim(), selected)
		rightCell = pad(d.theme.dim().Render(fmt.Sprintf("%d archived", s.Archived)), rightW)
	}
	line := d.theme.cursorMark(selected) +
		pad(styleMatches(s.Name, m.head, nameStyle), d.nameW) + "  " +
		pad(styleMatches(s.Title, m.tail, boldWhen(lipgloss.NewStyle(), selected)), max(w-fixed, minTitleWidth)) + "  " +
		rightCell
	return ansi.Truncate(line, w, "…")
}

// archivedLine renders one archived task row: dim id, title, and relative age
// only — no status or gate columns, since an archived task is done by
// convention. The row stays selectable so enter opens its read-only preview.
func (d compactDelegate) archivedLine(r Row, selected bool, w int, m matches) string {
	fixed := 2 + d.idW + 2
	if d.showAge {
		fixed += 2 + d.ageW
	}
	dim := boldWhen(d.theme.dim(), selected)
	line := d.theme.cursorMark(selected) +
		pad(styleMatches(r.ID, m.head, dim), d.idW) + "  " +
		pad(styleMatches(r.Title, m.tail, dim), max(w-fixed, minTitleWidth))
	if d.showAge {
		line += "  " + dim.Render(pad(ageCell(r, d.now), d.ageW))
	}
	return ansi.Truncate(line, w, "…")
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
func (d compactDelegate) taskLine(r Row, selected bool, w int, m matches) string {
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
		pad(styleMatches(r.ID, m.head, boldWhen(d.theme.accent(), selected)), d.idW) + "  " +
		pad(styleMatches(r.Title, m.tail, boldWhen(lipgloss.NewStyle(), selected)), max(w-fixed, minTitleWidth)) + "  " +
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

func (d compactDelegate) statusCell(r Row, selected bool) string {
	cell := boldWhen(d.theme.statusStyle(r.Status), selected).Render(pad(r.Status, statusColWidth))
	if inProgress(r) && d.glyph != "" {
		cell += " " + d.glyph
	}
	return cell
}

func ageCell(r Row, now time.Time) string {
	if r.UpdatedAt.IsZero() {
		return ""
	}
	return humanize.RelTime(r.UpdatedAt, now)
}

func maxAgeWidth(sections []Section, now time.Time) int {
	w := 0
	for _, s := range sections {
		for _, r := range s.Rows {
			w = max(w, lipgloss.Width(ageCell(r, now)))
		}
	}
	return w
}
